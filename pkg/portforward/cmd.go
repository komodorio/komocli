package portforward

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const flagJWT = "jwt"
const flagTimeout = "timeout"
const flagBrowser = "browser"
const flagAddress = "address"

var Command = &cobra.Command{
	// komocli port-forward <agentId> <namespace/pod:port> [local-port]
	Use:     "port-forward",
	Short:   "Starts port forwarding client process",
	Example: "komocli port-forward <agentId> <namespace/pod:port> [local-port]",
	Args:    cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		jwt, err := cmd.Flags().GetString(flagJWT)
		if err != nil {
			return err
		}

		timeout, err := cmd.Flags().GetDuration(flagTimeout)
		if err != nil {
			return err
		}

		browser, err := cmd.Flags().GetBool(flagBrowser)
		if err != nil {
			return err
		}

		address, err := cmd.Flags().GetString(flagAddress)
		if err != nil {
			return err
		}

		localPort := ""
		if len(args) > 2 {
			localPort = args[2]
		}

		return run(cmd.Context(), args[0], args[1], localPort, jwt, timeout, browser, address)
	},
}

func init() {
	Command.Flags().Duration(flagTimeout, 5*time.Second, "Timeout for operations")
	Command.Flags().String(flagJWT, "", "JWT Authentication token")
	Command.Flags().String(flagAddress, "localhost", "Network address to listen on (aka 'bind address')")
	Command.Flags().Bool(flagBrowser, false, "Open forwarded address automatically in browser")
	err := Command.MarkFlagRequired(flagJWT)
	if err != nil {
		panic(err)
	}
}

func run(ctx context.Context, agent string, remote string, local string, jwt string, timeout time.Duration, browserOpen bool, address string) error {
	rSpec := RemoteSpec{
		AgentId: agent,
	}

	parts := strings.Split(remote, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for remote namespace/podName:port, missing '/'")
	}
	rSpec.Namespace = parts[0]

	parts = strings.Split(parts[1], ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for remote namespace/pod:port, missing ':'")
	}
	rSpec.PodName = parts[0]

	var err error
	rSpec.RemotePort, err = strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("failed to parse remote port: %w", err)
	}

	lport := rSpec.RemotePort
	if local != "" {
		lport, err = strconv.Atoi(local)
		if err != nil {
			return fmt.Errorf("failed to parse local port: %s", err)
		}
	}

	if jwt == "" {
		jwt = os.Getenv("KOMOCLI_JWT")
	}

	ctl := NewController(rSpec, address, lport, jwt, timeout)

	afterInit := func() {
		if browserOpen {
			time.Sleep(250 * time.Millisecond)
			url := fmt.Sprintf("http://%s:%d", address, lport) // https would not work well anyway
			log.Infof("Opening in browser: %s", url)
			err := browser.OpenURL(url)
			if err != nil {
				log.Warnf("Failed to open Web browser: %s", err)
			}
		}
	}

	err = ctl.Run(ctx, afterInit)
	if err != nil {
		return fmt.Errorf("error while trying to forward port: %w", err)
	}

	return nil
}
