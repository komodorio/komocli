package portforward

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"sync"
)

type Controller struct {
	RemoteSpec RemoteSpec
	LocalPort  int
	Token      string
}

func (c *Controller) Run(ctx context.Context) error {
	// template message for session starts
	initMsg := &SessionMessage{
		MessageType: MTPortForwardInit,
		Data: &WSPortForwardInitData{
			Namespace: c.RemoteSpec.Namespace,
			PodName:   c.RemoteSpec.PodName,
			Port:      c.RemoteSpec.RemotePort,
		},
	}

	err := c.testConnection(ctx, initMsg)
	if err != nil {
		return err
	}
	log.Infof("Finished testing the connectivity, ready to accept connections")

	// check and bind local port, mind the host
	host := os.Getenv("KOMOCLI_BIND")
	if host == "" {
		host = "localhost"
	}
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, c.LocalPort))
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		log.Debugf("Stopping to accept connections")
		listen.Close()
	}()

	// setup connection handler
	c.acceptIncomingConns(ctx, listen, initMsg)

	// if not errored, shut down open conns gracefully
	return nil
}

func (c *Controller) testConnection(ctx context.Context, initMsg *SessionMessage) error {
	// test connect to Komodor WS endpoint
	ws := NewWSConnectionWrapper(ctx, nil, c.RemoteSpec.AgentId, c.Token, true, *initMsg)
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

func (c *Controller) acceptIncomingConns(ctx context.Context, listen net.Listener, initMsg *SessionMessage) {
	wg := sync.WaitGroup{}
	conns := []*WSConnectionWrapper{}
	for {
		conn, err := listen.Accept()
		if err != nil {
			if !isConnClosedErr(err) {
				log.Warnf("Failed to accept incoming connection: %+v", err)
			}
			break
		}

		log.Infof("Accepted connection: %v", conn.LocalAddr())
		ws := NewWSConnectionWrapper(ctx, conn, c.RemoteSpec.AgentId, c.Token, false, *initMsg)
		conns = append(conns, ws)

		wg.Add(1)
		go func() {
			err := ws.Run()
			if err != nil {
				log.Warnf("Failed to run port-forwarding: %s", err)
			}

			err = ws.Stop()
			if err != nil {
				log.Warnf("Failed to stop port-forwarding: %s", err)
			}
			wg.Done()
		}()
	}
	log.Infof("Stopped accepting incoming connections")

	for _, ws := range conns {
		err := ws.Stop()
		if err != nil {
			log.Warnf("Failed to stop port-forwarding: %s", err)
		}
	}

	wg.Wait()
}

func NewController(rSpec RemoteSpec, lport int, jwt string) *Controller {
	return &Controller{
		RemoteSpec: rSpec,
		LocalPort:  lport,
		Token:      jwt,
	}
}

type RemoteSpec struct {
	AgentId    string
	Namespace  string
	PodName    string
	RemotePort int
}
