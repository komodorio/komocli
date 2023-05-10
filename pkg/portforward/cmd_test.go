package portforward

import (
	"context"
	"github.com/spf13/cobra"
	"os"
	"testing"
)

func TestParams(t *testing.T) {
	cases := []struct {
		args       []string
		shouldFail bool
		remote     int
		local      int
	}{
		{
			args:       []string{},
			shouldFail: true,
			remote:     0,
			local:      0,
		},
		{
			args:       []string{"test"},
			shouldFail: true,
			remote:     0,
			local:      0,
		},
		{
			args:       []string{"", "1"},
			shouldFail: false,
			remote:     1,
			local:      1,
		},
		{
			args:       []string{"", ":2"},
			shouldFail: false,
			remote:     2,
			local:      0,
		},
		{
			args:       []string{"", "3:4"},
			shouldFail: false,
			remote:     4,
			local:      3,
		},
	}

	for _, c := range cases {
		params := CmdParams{}
		cmd := &cobra.Command{}
		setupFlags(cmd)
		err := validateFlags(cmd)
		if err != nil {
			t.Fatal(err)
		}

		err = params.AcceptArgs(cmd, c.args)

		if err != nil && !c.shouldFail {
			t.Errorf("test case is expected to fail: %v", c)
		} else {
			if params.LocalPort != c.local || params.RemotePort != c.remote {
				t.Errorf("wrong port numbers in test case: %v", c)
			}
		}
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
	cmd := NewCommand()
	err := cmd.Execute()
	if err != nil {
		t.Logf("We expect it to show help and return error: %v", err)
	}
}
