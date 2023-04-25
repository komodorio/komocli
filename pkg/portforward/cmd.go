package portforward

import (
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"os"
	"strconv"
	"strings"
	"time"
)

const flagJWT = "jwt"
const flagTimeout = "timeout"

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

		return run(cmd.Context(), args[0], args[1], args[2], jwt, timeout)
	},
}

func init() {
	Command.Flags().String(flagJWT, "", "JWT Authentication token")
	Command.Flags().Duration(flagTimeout, 5*time.Second, "Timeout for operations")
}

func run(ctx context.Context, agent string, remote string, local string, jwt string, timeout time.Duration) error {
	rSpec := RemoteSpec{
		AgentId: agent,
	}

	parts := strings.Split(remote, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for remote namespace/podName:port")
	}
	rSpec.Namespace = parts[0]

	parts = strings.Split(parts[1], ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for remote namespace/pod:port")
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
			return fmt.Errorf("failed to parse local port")
		}
	}

	if jwt == "" {
		jwt = os.Getenv("KOMOCLI_JWT")
	}

	ctl := NewController(rSpec, lport, jwt, timeout)
	err = ctl.Run(ctx)
	if err != nil {
		return fmt.Errorf("error while trying to forward port: %w", err)
	}

	return nil
}
