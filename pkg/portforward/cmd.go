package portforward

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const flagToken = "token"
const flagTimeout = "timeout"
const flagBrowser = "browser"
const flagAddress = "address"
const flagNamespace = "namespace"
const flagCluster = "cluster"

type CmdParams struct {
	Namespace    string
	Token        string
	Timeout      time.Duration
	OpenBrowser  bool
	Address      string
	Cluster      string
	LocalPort    int
	RemotePort   int
	ResourceName string
}

func (o *CmdParams) AcceptArgs(cmd *cobra.Command, args []string) (err error) {
	if len(args) != 2 {
		return errors.New("exactly two arguments required for command")
	}

	o.ResourceName = args[0]

	o.LocalPort, o.RemotePort, err = splitPort(args[1])
	if err != nil {
		return err
	}

	flags := cmd.Flags()
	o.Token, err = flags.GetString(flagToken)
	if err != nil {
		return err
	}

	if o.Token == "" {
		o.Token = os.Getenv("KOMOCLI_JWT")
	}

	o.Timeout, err = flags.GetDuration(flagTimeout)
	if err != nil {
		return err
	}

	o.OpenBrowser, err = flags.GetBool(flagBrowser)
	if err != nil {
		return err
	}

	o.Address, err = flags.GetString(flagAddress)
	if err != nil {
		return err
	}

	o.Namespace, err = flags.GetString(flagNamespace)
	if err != nil {
		return err
	}

	o.Cluster, err = flags.GetString(flagCluster)
	if err != nil {
		return err
	}

	return nil
}

func (o *CmdParams) Run(ctx context.Context) (err error) {
	rSpec := RemoteSpec{
		AgentId: o.Cluster,
	}

	rSpec.Namespace = o.Namespace
	rSpec.PodName = o.ResourceName

	ctl := NewController(rSpec, o.Address, o.LocalPort, o.Token, o.Timeout)

	afterInit := func() {
		if o.OpenBrowser {
			url := fmt.Sprintf("http://%s:%d", o.Address, o.LocalPort) // https would not work well anyway
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

func NewCommand() *cobra.Command {
	var cmd = &cobra.Command{
		// komocli port-forward <agentId> <namespace/pod:port> [local-port]
		Use:     "port-forward",
		Short:   "Starts port forwarding client process",
		Example: "komocli port-forward <agentId> <namespace/pod:port> [local-port]",
		Args:    cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			opts := CmdParams{}
			err := opts.AcceptArgs(c, args)
			if err != nil {
				return err
			}

			return opts.Run(c.Context())
		},
	}

	setupFlags(cmd)
	err := validateFlags(cmd)
	if err != nil {
		panic(err)
	}

	return cmd
}

func setupFlags(cmd *cobra.Command) {
	cmd.Flags().Duration(flagTimeout, 5*time.Second, "Timeout for operations")
	cmd.Flags().String(flagToken, "", "JWT Authentication token")
	cmd.Flags().String(flagAddress, "localhost", "Network address to listen on (aka 'bind address')")
	cmd.Flags().Bool(flagBrowser, false, "Open forwarded address automatically in browser")
	cmd.Flags().String(flagNamespace, "default", "Namespace for the resource")
	cmd.Flags().String(flagCluster, "", "Komodor cluster name that contains resource")
}

func validateFlags(cmd *cobra.Command) error {
	err := cmd.MarkFlagRequired(flagToken)
	if err != nil {
		return err
	}

	err = cmd.MarkFlagRequired(flagCluster)
	if err != nil {
		return err
	}

	err = cmd.MarkFlagRequired(flagNamespace)
	if err != nil {
		return err
	}
	return nil
}

func splitPort(port string) (local, remote int, err error) {
	// logic copied from kubectl code portforward.go

	parts := strings.Split(port, ":")
	if parts[0] != "" {
		local, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
	}

	if len(parts) == 2 {
		remote, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}

		return local, remote, nil
	}

	return local, local, nil
}
