package portforward

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strconv"
)

func RunPortForwarding(ctx context.Context, agentId string, ns string, pod string, rPort int, lport int, jwt string) error {
	// template message for session starts
	initMsg := &SessionMessage{
		MessageType: MTPodExecInit,
		Data: &PodExecInitData{
			Namespace: ns,
			PodName:   pod,
			Cmd:       PortForwardCMDPrefix + strconv.Itoa(rPort),
		},
	}

	err := testConnection(ctx, agentId, jwt, initMsg)
	if err != nil {
		return err
	}
	log.Infof("Finished testing the connectivity, ready to accept connections")

	// check and bind local port, mind the host
	host := os.Getenv("KOMOCLI_BIND")
	if host == "" {
		host = "localhost"
	}
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, lport))
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		listen.Close()
	}()

	// setup connection handler
	acceptIncomingConns(ctx, listen, agentId, jwt, initMsg)

	<-ctx.Done() // chill on ctx

	// if not errored, shut down open conns gracefully
	return nil
}

func testConnection(ctx context.Context, agentId string, jwt string, initMsg *SessionMessage) error {
	// test connect to Komodor WS endpoint
	ws := NewWSConnectionWrapper(ctx, nil, agentId, jwt, true, *initMsg)
	err := ws.Run()
	if err != nil {
		log.Warnf("Failed to test port-forward operability: %+v", err)
		return err
	}

	err = ws.Stop()
	if err != nil {
		log.Warnf("Failed to send session termination message: %s", err)
		return err
	}
	return err
}

func acceptIncomingConns(ctx context.Context, listen net.Listener, agentId string, jwt string, initMsg *SessionMessage) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Warnf("Failed to accept incoming connection: %+v", err)
			break
		}

		log.Infof("Accepted connection: %v", conn.LocalAddr())
		ws := NewWSConnectionWrapper(ctx, conn, agentId, jwt, false, *initMsg)
		go func() {
			<-ctx.Done()
			ws.Stop()
		}()

		go func() {
			ws.Run()
		}()
	}
	log.Infof("Stopped accepting incoming connections")
}
