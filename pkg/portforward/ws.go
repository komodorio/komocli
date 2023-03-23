package portforward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"os"
)

type WSConnectionWrapper struct {
	ctx        context.Context
	tcpConn    net.Conn
	wsConn     *websocket.Conn
	agentId    string
	jwt        string
	isConnTest bool
}

func (w *WSConnectionWrapper) Run() {
	defer func() {
		if !w.isConnTest {
			log.Infof("Done working with connection: %v", w.tcpConn.LocalAddr())
			_ = w.tcpConn.Close()
		} else {
			log.Infof("Finished testing the connectivity")
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
		return
	}

	// write initial msg
	msg := SessionMessage{
		SessionId:   PortForwardCMDPrefix + uuid.New().String(),
		MessageId:   uuid.New().String(),
		MessageType: MTPodExecInit,
	}

	txt, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Failed to serialize output message: %s", err)
		return
	}

	err = w.wsConn.WriteMessage(websocket.TextMessage, txt)
	if err != nil {
		log.Errorf("Failed to send output message over WS: %s", err)
		return
	}

	// 		setup read/write loops
	bridge := NewWSBridge(w.wsConn)
	go func() {
		if w.isConnTest {
			log.Debugf("Not trying to send data due to validation loop")
			return
		}

		n, err := io.Copy(bridge, w.tcpConn)
		log.Infof("Done tcp->ws transfer: %d bytes", n)
		if err != nil {
			log.Warnf("Problems transfering tcp->ws: %s", err)
		}
	}()

	var wr io.Writer
	if w.isConnTest {
		wr = &bytes.Buffer{}
	} else {
		wr = w.tcpConn
	}
	n, err := io.Copy(wr, bridge)
	log.Infof("Done ws->tcp transfer: %d bytes", n)
	if err != nil {
		log.Warnf("Problems in ws->tcp transfer: %s", err)
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

func NewWSConnectionWrapper(ctx context.Context, conn net.Conn, agentId string, jwt string, isConnTest bool) *WSConnectionWrapper {
	return &WSConnectionWrapper{
		ctx:        ctx,
		tcpConn:    conn,
		isConnTest: isConnTest,

		agentId: agentId,
		jwt:     jwt,
	}
}
