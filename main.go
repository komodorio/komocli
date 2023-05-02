package main

import (
	"context"
	"fmt"
	"github.com/komodorio/komocli/pkg/portforward"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	version = "0.0.0"
	commit  = "none"
	date    = "unknown"
)

const flagVerbose = "verbose"

var rootCtxCancel context.CancelFunc = func() {}
var RootCmd = &cobra.Command{
	Use:           filepath.Base(os.Args[0]),
	Version:       version,
	Short:         "Komodor CLI",
	Long:          `Allows interacting with Komodor platform for automation`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := os.Setenv("KOMOCLI_VERSION", version) // for anyone willing to access it
		if err != nil {
			fmt.Println("Failed to remember app version because of error: " + err.Error())
		}

		if verbose, err := cmd.Flags().GetBool(flagVerbose); err == nil {
			verbose = verbose || os.Getenv("DEBUG") != ""
			setupLogging(verbose)
		} else {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		rootCtxCancel = cancel

		cmd.SetContext(ctx)

		osSignal := make(chan os.Signal, 1)
		signal.Notify(osSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			oscall := <-osSignal
			log.Warnf("Stopping on signal: %s\n", oscall)
			rootCtxCancel()
		}()

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		rootCtxCancel()
	},
}

func init() {
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show verbose debug information and logging")

	RootCmd.AddCommand(portforward.Command)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatalf("Failed running CLI: %s", err)
	}

	log.Infof("Done.")
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
