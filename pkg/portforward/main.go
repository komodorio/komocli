package portforward

import "context"

func RunPortForwarding(ctx context.Context, ns string, pod string, rPort int, lport int, jwt string) {
	// test connect to Komodor WS endpoint

	// check and bind local port, mind the host

	// setup connection handler

	// setup read/write loops

	// chill on ctx
	<-ctx.Done()

	// if not errored,
}
