package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
	"github.com/slackpad/consul-live/commands"
)

func main() {
	// This helps us find our logs vs. those from Consul running under our
	// control.
	log.SetPrefix("@@@ ==> ")

	c := cli.NewCLI("consul-live", "0.0.1")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"block":      commands.BlockCommandFactory,
		"cluster":    commands.ClusterCommandFactory,
		"federation": commands.FederationCommandFactory,
		"kill":       commands.KillCommandFactory,
		"load":       commands.LoadCommandFactory,
		"upgrade":    commands.UpgradeCommandFactory,
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}
	os.Exit(exitStatus)
}
