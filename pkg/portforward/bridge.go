package portforward

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// WSBridge implements both Reader and Writer that translate data from WS messages into TCP data and back
type WSBridge struct {
	ws *websocket.Conn
}

func (b *WSBridge) Write(p []byte) (n int, err error) {
	// we received data via TCP and now want to translate it into WS message

	// loop bridged messages
	return len(p), err
}

func (b *WSBridge) Read(p []byte) (n int, err error) {
	_, bytes, err := b.ws.ReadMessage()
	if err != nil {
		log.Warnf("Failed to read message from WS: %s", err)
	} else {
		log.Debugf("Read msg over WS: %s", bytes)
	}
	return len(bytes), err
}

func NewWSBridge(ws *websocket.Conn) *WSBridge {
	return &WSBridge{
		ws: ws,
	}
}
