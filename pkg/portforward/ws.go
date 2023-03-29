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
	"sync"
	"time"
)

type WSConnectionWrapper struct {
	ctx        context.Context
	tcpConn    net.Conn
	wsConn     *websocket.Conn
	agentId    string
	jwt        string
	isConnTest bool
	SessionId  string
	initMsg    *SessionMessage

	chReady  chan struct{}
	graceful bool
	mx       sync.Mutex
	closed   bool
	readBuf  bytes.Buffer
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
		base = "wss://app.komodor.com"
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

	// write initial msg
	w.initMsg.MessageId = uuid.New().String()
	w.initMsg.Timestamp = time.Now()

	err = w.sendWS(w.initMsg)
	if err != nil {
		return err
	}

	// write loop
	go func() {
		if w.isConnTest {
			log.Debugf("Not trying to send data due to validation loop")
			return
		}

		log.Debugf("Starting tcp->ws transfer")
		n, err := io.Copy(w, w.tcpConn)
		log.Infof("Done tcp->ws transfer: %d bytes", n)
		if err != nil {
			if w.graceful {
				log.Debugf("Result of transfering tcp->ws: %s", err)
			} else {
				log.Warnf("Problems transfering tcp->ws: %s", err)
			}
		}
	}()

	// read loop
	var wr io.Writer
	if w.isConnTest {
		wr = &bytes.Buffer{}
	} else {
		wr = w.tcpConn
	}
	n, err := io.Copy(wr, w)
	log.Infof("Done ws->tcp transfer: %d bytes", n)
	if err != nil {
		log.Warnf("Problems in ws->tcp transfer: %s", err)
		return err
	}

	return nil
}

func (w *WSConnectionWrapper) sendWS(msg *SessionMessage) error {
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
	return nil
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

func (w *WSConnectionWrapper) Write(p []byte) (n int, err error) {
	<-w.chReady // we need to wait for ack before writing anything

	// we received data via TCP and now want to translate it into WS message
	msg := SessionMessage{
		MessageId:   uuid.New().String(),
		SessionId:   w.SessionId,
		MessageType: MTStdin,
		Data: &PodExecStdinData{
			Input: base64.StdEncoding.EncodeToString(p),
		},
		Timestamp: time.Now(),
	}

	err = w.sendWS(&msg)
	if err != nil {
		return 0, err
	}

	// loop bridged messages
	return len(p), err
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
			err2 := w.readWS()
			if err2 != nil {
				err = err2
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
		log.Warnf("Failed to read message from WS: %s", err)
		return err
	}

	log.Debugf("Read msg over WS: %s", bts)
	var msg SessionMessage
	err = json.Unmarshal(bts, &msg)
	if err != nil {
		return err
	}

	if msg.MessageType == MTAck && msg.Data.(*PodExecAckData).AckedMessageID == w.initMsg.MessageId {
		w.SessionId = msg.SessionId
		close(w.chReady) // ready to write data into WS
	}

	switch msg.MessageType {
	case MTStdout:
		payload, err := base64.StdEncoding.DecodeString(msg.Data.(*PodExecStdoutData).Out)
		if err != nil {
			err := w.sendWS(&SessionMessage{
				MessageId:   uuid.NewString(),
				SessionId:   w.SessionId,
				MessageType: MTError,
				Data: &PodExecErrorData{
					OriginalMessageID: msg.MessageId,
					ErrorMessage:      fmt.Sprintf("Failed to decode Base64: %s", err),
				},
				Timestamp: time.Now(),
			})
			if err != nil {
				log.Debugf("Failed to send WS err: %s", err)
			}
		} else {
			w.readBuf.Write(payload)
		}
	case MTAck:
		if w.isConnTest {
			err = io.EOF // enough for connection test
		} // ok otherwise
	case MTError:
		err = fmt.Errorf("received error from remote: %s", msg.Data.(*PodExecErrorData).ErrorMessage)
	case MTTermination:
		log.Infof("Got termination message, gotta shutdown")
		w.graceful = true
		err = io.EOF
	default:
		log.Warnf("Unhandled WS message: %s", msg)
	}

	return err
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
	err := w.sendWS(&SessionMessage{
		MessageId:   uuid.NewString(),
		SessionId:   w.SessionId,
		MessageType: MTTermination,
		Data: &PodExecSessionTerminationData{
			ProcessExitCode: 0,
			ExitMessage:     "Stopping",
		},
		Timestamp: time.Now(),
	})
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

	return nil
}

func NewWSConnectionWrapper(ctx context.Context, conn net.Conn, agentId string, jwt string, isConnTest bool, initMsg SessionMessage) *WSConnectionWrapper {
	return &WSConnectionWrapper{
		ctx:        ctx,
		tcpConn:    conn,
		isConnTest: isConnTest,
		initMsg:    &initMsg, // this is intentional to accept dereferenced value, to create a copy of it

		agentId: agentId,
		jwt:     jwt,

		chReady: make(chan struct{}),
	}
}
