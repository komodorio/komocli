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

func (ws *WSConnectionWrapper) Run() error {
	defer func() {
		if !ws.isConnTest {
			log.Infof("Done working with connection: %v", ws.tcpConn.LocalAddr())
			_ = ws.tcpConn.Close()
		}
	}()

	base := os.Getenv("KOMOCLI_WS_URL")
	if base == "" {
		base = DefaultWSAddress
	}

	hdr := http.Header{}
	url := fmt.Sprintf("%s/ws/client/%s", base, ws.agentId)

	if os.Getenv("KOMOCLI_DEV") == "" {
		c := http.Cookie{Name: "JWT_TOKEN", Value: ws.jwt}
		hdr.Set("Cookie", c.String())
	} else {
		url += "?authorization=" + ws.jwt
	}

	var err error
	ws.wsConn, err = ws.connectWS(url, hdr)
	if err != nil {
		log.Warnf("Failed to open WebSocket connection: %+v", err)
		return err
	}

	err = ws.init()
	if err != nil {
		return err
	}

	readingDone := make(chan struct{})
	writingDone := make(chan error)

	go ws.writeLoop(readingDone)
	go ws.readLoop(writingDone)
	go ws.loopKeepAlive()

	select { // wait either
	case <-ws.ctx.Done():
		err = ws.ctx.Err()
	case e := <-writingDone:
		err = e
	case <-readingDone:
	}

	e := ws.Stop()
	if err == nil { // don't mask previous error
		err = e
	}

	return err
}

func (ws *WSConnectionWrapper) init() error {
	// write initial msg
	ws.initMsg.MessageId = uuid.New().String()
	ws.initMsg.Timestamp = time.Now()

	return ws.sendWS(ws.initMsg, true)
}

func (ws *WSConnectionWrapper) writeLoop(readingDone chan struct{}) {
	// write loop
	if ws.isConnTest {
		log.Debugf("Not trying to send data due to validation loop")
		return
	}

	log.Debugf("Starting tcp->ws transfer")
	n, err := io.Copy(ws, ws.tcpConn)
	log.Infof("Done tcp->ws transfer: %d bytes", n)
	if err != nil && !isConnClosedErr(err) {
		log.Warnf("Problems transfering tcp->ws: %s", err)
	}
	close(readingDone)
}

func (ws *WSConnectionWrapper) readLoop(writingDone chan error) {
	// read loop
	var wr io.Writer
	if ws.isConnTest {
		wr = &bytes.Buffer{}
	} else {
		wr = ws.tcpConn
	}
	n, err := io.Copy(wr, ws)
	log.Infof("Done ws->tcp transfer: %d bytes", n)
	if err != nil && !isConnClosedErr(err) {
		log.Warnf("Problems in ws->tcp transfer: %s", err)
		writingDone <- err
	}
	close(writingDone)
}

func (ws *WSConnectionWrapper) loopKeepAlive() {
	if !ws.isConnTest {
		return // no keepalive for conn test
	}

	ticker := time.NewTicker(5 * time.Second) // TODO: parameterize it?
	defer ticker.Stop()

	for {
		_, ok := <-ticker.C
		if !ok || ws.closed { // if it is stopped
			break
		}

		select {
		case <-ws.ctx.Done():
			break
		default:
		}

		err := ws.sendWS(ws.newSessMessage(MTKeepAlive, &WSKeepaliveData{}), true)
		if err != nil {
			log.Errorf("Failed to send keep-alive message: %s", err)
			err := ws.Stop()
			if err != nil {
				log.Warnf("Failed to stop session: %s", err)
			}
			break
		}
	}

	log.Debugf("KeepAlive loop done")
}

func (ws *WSConnectionWrapper) sendWS(msg *SessionMessage, needsAck bool) error {
	txt, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Failed to serialize output message: %s", err)
		return err
	}

	log.Debugf("Sending WS message: %s", txt)
	err = ws.wsConn.WriteMessage(websocket.TextMessage, txt)
	if err != nil {
		log.Errorf("Failed to send output message over WS: %s", err)
		return err
	}

	if needsAck {
		ctx, cancel := context.WithTimeout(ws.ctx, ws.timeout) // TODO: save cancelfn, too?
		ws.pendingAckMessages[msg.MessageId] = cancel
		go ws.expectAck(ctx, msg)
	}

	return nil
}

func (ws *WSConnectionWrapper) expectAck(ctx context.Context, msg *SessionMessage) {
	<-ctx.Done()
	if cancel, found := ws.pendingAckMessages[msg.MessageId]; found {
		cancel()
		if ctx.Err() != nil {
			log.Warnf("Did not receive ack within timeout for message %s: %s", msg.MessageId, ctx.Err())
			err := ws.Stop()
			if err != nil {
				log.Warnf("Failed to stop session: %s", err)
			}
		}
	}
}

func (ws *WSConnectionWrapper) connectWS(url string, hdr http.Header) (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer
	log.Infof("Connecting to WS backend at %s", url)
	conn, resp, err := dialer.DialContext(ws.ctx, url, hdr)
	if err != nil {
		if resp != nil {
			log.Errorf("handshake failed with status %d", resp.StatusCode)
		}
		return nil, err
	}
	return conn, nil
}

func (ws *WSConnectionWrapper) Write(b []byte) (n int, err error) {
	<-ws.chReady // we need to wait for ack before writing anything

	// we received data via TCP and now want to translate it into WS message
	msg := ws.newSessMessage(MTStdin, &WSStdinData{
		Input: base64.StdEncoding.EncodeToString(b),
	})

	err = ws.sendWS(msg, true)
	if err != nil {
		return 0, err
	}

	// loop bridged messages
	return len(b), err
}

func (ws *WSConnectionWrapper) Read(b []byte) (int, error) {
	// read from pushed msg into b
	n, err := ws.readBuf.Read(b)
	if err == io.EOF {
		if !ws.closed {
			err = nil // let it just finish the iteration
		}

		if n == 0 {
			// wait for the next chunk to arrive
			readWSErr := ws.readWS()
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

func (ws *WSConnectionWrapper) readWS() error {
	_, bts, err := ws.wsConn.ReadMessage()
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

	if ws.isInitAck(&msg) {
		ws.SessionId = msg.SessionId
		close(ws.chReady) // ready to write data into WS
	}

	return ws.handleMsg(&msg)
}

func (ws *WSConnectionWrapper) isInitAck(msg *SessionMessage) bool {
	return msg.MessageType == MTAck && msg.Data.(*WSAckData).AckedMessageID == ws.initMsg.MessageId
}

func (ws *WSConnectionWrapper) handleMsg(msg *SessionMessage) error {
	switch msg.MessageType {
	case MTStdout:
		ws.receiveOutput(msg)
	case MTAck:
		return ws.handleMsgAck(msg)
	case MTError:
		return fmt.Errorf("received error from remote: %s", msg.Data.(*WSErrorData).ErrorMessage)
	case MTTermination:
		log.Infof("Got termination message, gotta shutdown")
		ws.graceful = true
		return io.EOF
	default:
		log.Warnf("Unhandled WS message: %s", msg)
	}
	return nil
}

func (ws *WSConnectionWrapper) handleMsgAck(msg *SessionMessage) error {
	var err error
	if ws.isConnTest {
		err = io.EOF // enough for connection test
	}

	acked := msg.Data.(*WSAckData).AckedMessageID

	if _, ok := ws.pendingAckMessages[acked]; ok {
		delete(ws.pendingAckMessages, acked)
	} else {
		log.Warnf("Received ack for unexpected message ID: %s", acked)
	}
	return err
}

func (ws *WSConnectionWrapper) receiveOutput(msg *SessionMessage) {
	payload, err := base64.StdEncoding.DecodeString(msg.Data.(*WSStdoutData).Out)
	if err != nil {
		log.Debugf("Failed to decode Base64: %s", err)
		err := ws.sendWS(ws.newSessMessage(MTError, &WSErrorData{
			OriginalMessageID: msg.MessageId,
			ErrorMessage:      fmt.Sprintf("Failed to decode Base64: %s", err),
		}), false)
		if err != nil {
			log.Debugf("Failed to send WS err: %s", err)
		}
	} else {
		ws.readBuf.Write(payload)
	}
}

func (ws *WSConnectionWrapper) Stop() error {
	ws.mx.Lock()
	defer ws.mx.Unlock()

	if ws.closed {
		log.Debugf("Already stopped")
		return nil
	}
	ws.closed = true

	log.Infof("Closing forwarded connection: %v", ws.tcpConn)
	err := ws.sendWS(ws.newSessMessage(MTTermination, &WSSessionTerminationData{
		ProcessExitCode: 0,
		ExitMessage:     "Stopping",
	}), false)
	if err != nil {
		log.Debugf("Failed to send WS termination: %s", err)
		return err
	}

	if !ws.isConnTest {
		err = ws.tcpConn.Close()
		if err != nil {
			log.Debugf("Failed to close connection: %s", err)
			return err
		}
	}

	return ws.wsConn.Close()
}

func (ws *WSConnectionWrapper) newSessMessage(t MessageType, payload interface{}) *SessionMessage {
	return &SessionMessage{
		MessageId:   uuid.NewString(),
		SessionId:   ws.SessionId,
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
