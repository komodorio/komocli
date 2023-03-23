package portforward

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
)

func RunPortForwarding(ctx context.Context, agentId string, ns string, pod string, rPort int, lport int, jwt string) error {
	// test connect to Komodor WS endpoint
	ws := NewWSConnectionWrapper(ctx, nil, agentId, jwt, true)
	ws.Run()

	// check and bind local port, mind the host
	host := os.Getenv("KOMOCLI_BIND")
	if host == "" {
		host = "localhost"
	}
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, lport))
	if err != nil {
		return err
	}
	defer listen.Close()

	// setup connection handler
	go acceptIncomingConn(ctx, listen, agentId, jwt)

	<-ctx.Done() // chill on ctx

	// if not errored, shut down open conns gracefully
	return nil
}

func acceptIncomingConn(ctx context.Context, listen net.Listener, agentId string, jwt string) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalf("Failed to accept incoming connection: %+v", err)
			break
		}

		log.Infof("Accepted connection: %v", conn.LocalAddr())
		ws := NewWSConnectionWrapper(ctx, conn, agentId, jwt, false)
		go ws.Run()
	}
}
