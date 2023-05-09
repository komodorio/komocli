package portforward

import (
	"context"
	"errors"
	"github.com/spf13/cobra"
	"os"
	"testing"
)

func TestParams(t *testing.T) {
	params := CmdParams{}
	cmd := &cobra.Command{}
	setupFlags(cmd)
	err := validateFlags(cmd)
	if err != nil {
		t.Fatal(err)
	}

	err = params.AcceptArgs(cmd, []string{})
	if err == nil {
		t.Fatal(errors.New("should fail when no params"))
	}

	err = params.AcceptArgs(cmd, []string{"test"})
	if err == nil {
		t.Fatal(errors.New("should fail when one param"))
	}

	err = params.AcceptArgs(cmd, []string{"", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if params.LocalPort != params.RemotePort || params.LocalPort != 1 {
		t.Fatal(errors.New("single port not handled correctly"))
	}

	params.LocalPort = 0
	err = params.AcceptArgs(cmd, []string{"", ":2"})
	if err != nil {
		t.Fatal(err)
	}
	if params.LocalPort != 0 || params.RemotePort != 2 {
		t.Fatal(errors.New("random local port not handled correctly"))
	}

	err = params.AcceptArgs(cmd, []string{"", "3:4"})
	if err != nil {
		t.Fatal(err)
	}
	if params.LocalPort != 3 || params.RemotePort != 4 {
		t.Fatal(errors.New("both ports not handled correctly"))
	}
}

func TestRun(t *testing.T) {
	err := os.Setenv("KOMOCLI_WS_URL", "ws:///")
	if err != nil {
		t.Fatal(err)
	}

	params := CmdParams{}

	err = params.Run(context.Background())
	if err != nil {
		t.Logf("We expect it to return error: %v", err)
	}
}

func TestNew(t *testing.T) {
	_ = NewCommand()
}
