package portforward

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strconv"
)

type Controller struct {
	AgentId    string
	Namespace  string
	PodName    string
	RemotePort int
	LocalPort  int
	Token      string
}

func (c *Controller) Run(ctx context.Context) error {
	// template message for session starts
	initMsg := &SessionMessage{
		MessageType: MTPodExecInit,
		Data: &PodExecInitData{
			Namespace: c.Namespace,
			PodName:   c.PodName,
			Cmd:       PortForwardCMDPrefix + strconv.Itoa(c.RemotePort),
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

	<-ctx.Done() // chill on ctx

	// if not errored, shut down open conns gracefully
	return nil
}

func (c *Controller) testConnection(ctx context.Context, initMsg *SessionMessage) error {
	// test connect to Komodor WS endpoint
	ws := NewWSConnectionWrapper(ctx, nil, c.AgentId, c.Token, true, *initMsg)
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
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Warnf("Failed to accept incoming connection: %+v", err)
			break
		}

		log.Infof("Accepted connection: %v", conn.LocalAddr())
		ws := NewWSConnectionWrapper(ctx, conn, c.AgentId, c.Token, false, *initMsg)
		go func() {
			<-ctx.Done()
			ws.Stop()
		}()

		go func() {
			err := ws.Run()
			if err != nil {
				log.Warnf("Failed to run port-forwarding: %s", err)
			}
		}()
	}
	log.Infof("Stopped accepting incoming connections")
}

func NewController(agentId string, ns string, pod string, rPort int, lport int, jwt string) *Controller {
	return &Controller{
		AgentId:    agentId,
		Namespace:  ns,
		PodName:    pod,
		RemotePort: rPort,
		LocalPort:  lport,
		Token:      jwt,
	}
}
