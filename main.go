package main

import (
	"context"
	"fmt"
	"github.com/komodorio/komocli/pkg/portforward"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var (
	version = "0.0.0"
	commit  = "none"
	date    = "unknown"
)

type options struct {
	Version bool `long:"version" description:"Show tool version"`
	Verbose bool `short:"v" long:"verbose" description:"Show verbose debug information"`

	JWT string `short:"t" long:"jwt" description:"JWT Authentication token"`
}

func main() {
	err := os.Setenv("KOMOCLI_VERSION", version) // for anyone willing to access it
	if err != nil {
		fmt.Println("Failed to remember app version because of error: " + err.Error())
	}

	opts, args := parseFlags()

	opts.Verbose = opts.Verbose || os.Getenv("DEBUG") != ""
	setupLogging(opts.Verbose)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		oscall := <-osSignal
		log.Warnf("Stopping on signal: %s\n", oscall)
		cancel()
	}()

	if args[0] == "port-forward" { // TODO: rework into cobra subcommands?
		// komocli port-forward <agentId> <namespace/pod:port> [local-port]
		rSpec := portforward.RemoteSpec{
			AgentId: args[1],
		}

		parts := strings.Split(args[2], "/")
		if len(parts) != 2 {
			log.Fatalf("Invalid format for remote namespace/podName:port")
		}
		rSpec.Namespace = parts[0]

		parts = strings.Split(parts[1], ":")
		if len(parts) != 2 {
			log.Fatalf("Invalid format for remote namespace/pod:port")
		}
		rSpec.PodName = parts[0]

		var err error
		rSpec.RemotePort, err = strconv.Atoi(parts[1])
		if err != nil {
			log.Fatalf("Failed to parse remote port")
		}

		lport := rSpec.RemotePort
		if len(args) >= 4 {
			lport, err = strconv.Atoi(args[3])
			if err != nil {
				log.Fatalf("Failed to parse local port")
			}
		}

		jwt := opts.JWT
		if jwt == "" {
			jwt = os.Getenv("KOMOCLI_JWT")
		}

		f := portforward.NewController(rSpec, lport, jwt) // FIXME: very bad CLI interface!
		err = f.Run(ctx)
		if err != nil {
			log.Fatalf("Error while trying to forward port: %+v", err)
		}
	} else {
		log.Fatalf("Unsupported arguments provided")
	}

	log.Infof("Done.")
}

func parseFlags() (options, []string) {
	opts := options{}
	args, err := flags.Parse(&opts)
	if err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}

		// we rely on default behavior to print the problem inside `flags` library
		os.Exit(1)
	}

	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if len(args) < 1 {
		fmt.Println("The program requires at least 1 argument, see --help for usage")
		os.Exit(1)
	}
	return opts, args
}

func setupLogging(verbose bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
		log.Debugf("Debug logging is enabled")
	} else {
		log.SetLevel(log.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
	log.Infof("Komodor CLI, version %s (%s @ %s)", version, commit, date)
}
