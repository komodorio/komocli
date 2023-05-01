package portforward

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const DefaultWSAddress = "wss://app.komodor.com"

type WSConnectionWrapper struct {
	ctx        context.Context
	tcpConn    net.Conn
	wsConn     *websocket.Conn
	agentId    string
	jwt        string
	isConnTest bool
	SessionId  string
	initMsg    *SessionMessage

	chReady            chan struct{}
	graceful           bool
	mx                 sync.Mutex
	closed             bool
	readBuf            bytes.Buffer
	timeout            time.Duration
	pendingAckMessages map[string]context.CancelFunc
}

func (w *WSConnectionWrapper) Run() error {
	defer func() {
		if !w.isConnTest {
			log.Infof("Done working with connection: %v", w.tcpConn.LocalAddr())
			_ = w.tcpConn.Close()
		}
	}()

	base := os.Getenv("KOMOCLI_WS_URL")
	if base == "" {
		base = DefaultWSAddress
	}

	hdr := http.Header{}
	url := fmt.Sprintf("%s/ws/client/%s", base, w.agentId)

	if os.Getenv("KOMOCLI_DEV") == "" {
		c := http.Cookie{Name: "JWT_TOKEN", Value: w.jwt}
		hdr.Set("Cookie", c.String())
	} else {
		url += "?authorization=" + w.jwt
	}

	var err error
	w.wsConn, err = w.connectWS(url, hdr)
	if err != nil {
		log.Warnf("Failed to open WebSocket connection: %+v", err)
		return err
	}

	err = w.init()
	if err != nil {
		return err
	}

	readingDone := make(chan struct{})
	writingDone := make(chan error)

	go w.writeLoop(readingDone)
	go w.readLoop(writingDone)
	go w.loopKeepAlive()

	select { // wait either
	case <-w.ctx.Done():
		err = w.ctx.Err()
	case e := <-writingDone:
		err = e
	case <-readingDone:
	}

	e := w.Stop()
	if err == nil { // don't mask previous error
		err = e
	}

	return err
}

func (w *WSConnectionWrapper) init() error {
	// write initial msg
	w.initMsg.MessageId = uuid.New().String()
	w.initMsg.Timestamp = time.Now()

	return w.sendWS(w.initMsg, true)
}

func (w *WSConnectionWrapper) writeLoop(readingDone chan struct{}) {
	// write loop
	if w.isConnTest {
		log.Debugf("Not trying to send data due to validation loop")
		return
	}

	log.Debugf("Starting tcp->ws transfer")
	n, err := io.Copy(w, w.tcpConn)
	log.Infof("Done tcp->ws transfer: %d bytes", n)
	if err != nil && !isConnClosedErr(err) {
		log.Warnf("Problems transfering tcp->ws: %s", err)
	}
	close(readingDone)
}

func (w *WSConnectionWrapper) readLoop(writingDone chan error) {
	// read loop
	var wr io.Writer
	if w.isConnTest {
		wr = &bytes.Buffer{}
	} else {
		wr = w.tcpConn
	}
	n, err := io.Copy(wr, w)
	log.Infof("Done ws->tcp transfer: %d bytes", n)
	if err != nil && !isConnClosedErr(err) {
		log.Warnf("Problems in ws->tcp transfer: %s", err)
		writingDone <- err
	}
	close(writingDone)
}

func (w *WSConnectionWrapper) loopKeepAlive() {
	if !w.isConnTest {
		return // no keepalive for conn test
	}

	ticker := time.NewTicker(5 * time.Second) // TODO: parameterize it?
	defer ticker.Stop()

	for {
		_, ok := <-ticker.C
		if !ok || w.closed { // if it is stopped
			break
		}

		select {
		case <-w.ctx.Done():
			break
		default:
		}

		err := w.sendWS(w.newSessMessage(MTKeepAlive, &WSKeepaliveData{}), true)
		if err != nil {
			log.Errorf("Failed to send keep-alive message: %s", err)
			err := w.Stop()
			if err != nil {
				log.Warnf("Failed to stop session: %s", err)
			}
			break
		}
	}

	log.Debugf("KeepAlive loop done")
}

func (w *WSConnectionWrapper) sendWS(msg *SessionMessage, needsAck bool) error {
	txt, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Failed to serialize output message: %s", err)
		return err
	}

	log.Debugf("Sending WS message: %s", txt)
	err = w.wsConn.WriteMessage(websocket.TextMessage, txt)
	if err != nil {
		log.Errorf("Failed to send output message over WS: %s", err)
		return err
	}

	if needsAck {
		ctx, cancel := context.WithTimeout(w.ctx, w.timeout) // TODO: save cancelfn, too?
		w.pendingAckMessages[msg.MessageId] = cancel
		go w.expectAck(ctx, msg)
	}

	return nil
}

func (w *WSConnectionWrapper) expectAck(ctx context.Context, msg *SessionMessage) {
	<-ctx.Done()
	if cancel, found := w.pendingAckMessages[msg.MessageId]; found {
		cancel()
		if ctx.Err() != nil {
			log.Warnf("Did not receive ack within timeout for message %s: %s", msg.MessageId, ctx.Err())
			err := w.Stop()
			if err != nil {
				log.Warnf("Failed to stop session: %s", err)
			}
		}
	}
}

func (w *WSConnectionWrapper) connectWS(url string, hdr http.Header) (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer
	log.Infof("Connecting to WS backend at %s", url)
	conn, resp, err := dialer.DialContext(w.ctx, url, hdr)
	if err != nil {
		if resp != nil {
			log.Errorf("handshake failed with status %d", resp.StatusCode)
		}
		return nil, err
	}
	return conn, nil
}

func (w *WSConnectionWrapper) Write(b []byte) (n int, err error) {
	<-w.chReady // we need to wait for ack before writing anything

	// we received data via TCP and now want to translate it into WS message
	msg := w.newSessMessage(MTStdin, &WSStdinData{
		Input: base64.StdEncoding.EncodeToString(b),
	})

	err = w.sendWS(msg, true)
	if err != nil {
		return 0, err
	}

	// loop bridged messages
	return len(b), err
}

func (w *WSConnectionWrapper) Read(b []byte) (int, error) {
	// read from pushed msg into b
	n, err := w.readBuf.Read(b)
	if err == io.EOF {
		if !w.closed {
			err = nil // let it just finish the iteration
		}

		if n == 0 {
			// wait for the next chunk to arrive
			readWSErr := w.readWS()
			if readWSErr != nil {
				err = readWSErr
			} // othwerwise err=EOF
		}
	}

	if n > 0 {
		log.Debugf("Bridged ws->tcp: %d bytes", n)
	}

	return n, err
}

func (w *WSConnectionWrapper) readWS() error {
	_, bts, err := w.wsConn.ReadMessage()
	if err != nil {
		if !isConnClosedErr(err) {
			log.Warnf("Failed to read message from WS: %s", err)
		}
		return err
	}

	log.Debugf("Read msg over WS: %s", bts)
	var msg SessionMessage
	err = json.Unmarshal(bts, &msg)
	if err != nil {
		return err
	}

	isInitAck := msg.MessageType == MTAck && msg.Data.(*WSAckData).AckedMessageID == w.initMsg.MessageId
	if isInitAck {
		w.SessionId = msg.SessionId
		close(w.chReady) // ready to write data into WS
	}

	return w.handleMsg(&msg)
}

func (w *WSConnectionWrapper) handleMsg(msg *SessionMessage) error {
	switch msg.MessageType {
	case MTStdout:
		w.receiveOutput(msg)
	case MTAck:
		return w.handleMsgAck(msg)
	case MTError:
		return fmt.Errorf("received error from remote: %s", msg.Data.(*WSErrorData).ErrorMessage)
	case MTTermination:
		log.Infof("Got termination message, gotta shutdown")
		w.graceful = true
		return io.EOF
	default:
		log.Warnf("Unhandled WS message: %s", msg)
	}
	return nil
}

func (w *WSConnectionWrapper) handleMsgAck(msg *SessionMessage) error {
	var err error
	if w.isConnTest {
		err = io.EOF // enough for connection test
	}

	acked := msg.Data.(*WSAckData).AckedMessageID

	if _, ok := w.pendingAckMessages[acked]; ok {
		delete(w.pendingAckMessages, acked)
	} else {
		log.Warnf("Received ack for unexpected message ID: %s", acked)
	}
	return err
}

func (w *WSConnectionWrapper) receiveOutput(msg *SessionMessage) {
	payload, err := base64.StdEncoding.DecodeString(msg.Data.(*WSStdoutData).Out)
	if err != nil {
		log.Debugf("Failed to decode Base64: %s", err)
		err := w.sendWS(w.newSessMessage(MTError, &WSErrorData{
			OriginalMessageID: msg.MessageId,
			ErrorMessage:      fmt.Sprintf("Failed to decode Base64: %s", err),
		}), false)
		if err != nil {
			log.Debugf("Failed to send WS err: %s", err)
		}
	} else {
		w.readBuf.Write(payload)
	}
}

func (w *WSConnectionWrapper) Stop() error {
	w.mx.Lock()
	defer w.mx.Unlock()

	if w.closed {
		log.Debugf("Already stopped")
		return nil
	}
	w.closed = true

	log.Infof("Closing forwarded connection: %v", w.tcpConn)
	err := w.sendWS(w.newSessMessage(MTTermination, &WSSessionTerminationData{
		ProcessExitCode: 0,
		ExitMessage:     "Stopping",
	}), false)
	if err != nil {
		log.Debugf("Failed to send WS termination: %s", err)
		return err
	}

	if !w.isConnTest {
		err = w.tcpConn.Close()
		if err != nil {
			log.Debugf("Failed to close connection: %s", err)
			return err
		}
	}

	return w.wsConn.Close()
}

func (w *WSConnectionWrapper) newSessMessage(t MessageType, payload interface{}) *SessionMessage {
	return &SessionMessage{
		MessageId:   uuid.NewString(),
		SessionId:   w.SessionId,
		MessageType: t,
		Data:        payload,
		Timestamp:   time.Now(),
	}
}

func NewWSConnectionWrapper(ctx context.Context, conn net.Conn, agentId string, jwt string, isConnTest bool, initMsg SessionMessage, timeout time.Duration) *WSConnectionWrapper {
	return &WSConnectionWrapper{
		ctx:        ctx,
		tcpConn:    conn,
		isConnTest: isConnTest,
		initMsg:    &initMsg, // this is intentional to accept dereferenced value, to create a copy of it

		agentId: agentId,
		jwt:     jwt,

		chReady: make(chan struct{}),

		timeout:            timeout,
		pendingAckMessages: map[string]context.CancelFunc{},
	}
}

func isConnClosedErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}
