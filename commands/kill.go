package commands

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

func KillCommandFactory() (cli.Command, error) {
	return &Kill{}, nil
}

type Kill struct {
}

func (c *Kill) Help() string {
	helpText := `
Usage consul-live kill -token=<token>
`
	return strings.TrimSpace(helpText)
}

func (c *Kill) Synopsis() string {
	return "Kills the current leader once the cluster is stable"
}

func (c *Kill) Run(args []string) int {
	var token string
	cmdFlags := flag.NewFlagSet("kill", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.StringVar(&token, "token", "", "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	config := func() *api.Config {
		c := api.DefaultConfig()
		c.Token = token
		return c
	}
	if err := c.run(config); err != nil {
		log.Println(err)
		return 1
	}

	return 0
}

func (c *Kill) run(config func() *api.Config) error {
	client, err := api.NewClient(config())
	if err != nil {
		return err
	}

	// Set a default Autopilot configuration that makes recovery quicker.
	ap, err := client.Operator().AutopilotGetConfiguration(nil)
	if err != nil {
		return err
	}
	ap.ServerStabilizationTime = api.NewReadableDuration(1 * time.Second)
	if ok, err := client.Operator().AutopilotCASConfiguration(ap, nil); !ok || err != nil {
		return err
	}

	delay := func() {
		time.Sleep(2 * time.Second)
	}

	// Wait for the cluster to be able to take a fault and then ask the
	// current leader to leave.
	for {
		sh, err := client.Operator().AutopilotServerHealth(nil)
		if err != nil {
			log.Printf("Could not get cluster health (will retry): %v", err)
			delay()
			continue
		}

		if sh.FailureTolerance < 1 {
			log.Printf("Cluster can't tolerate a failure (will retry)")
			delay()
			continue
		}

		var leader string
		for _, server := range sh.Servers {
			if server.Leader {
				leader = server.Address
				break
			}
		}

		if leader == "" {
			log.Printf("Cluster doesn't have a leader (will retry)")
			delay()
			continue
		}

		log.Printf("Attempting to kill %q...", leader)
		c := config()
		host, _, err := net.SplitHostPort(leader)
		if err != nil {
			return err
		}
		c.Address = fmt.Sprintf("%s:%d", host, 8500)
		tc, err := api.NewClient(c)
		if err != nil {
			return err
		}
		if err := tc.Agent().Leave(); err != nil {
			log.Printf("Failed to leave for %q: %v", leader, err)
			delay()
			continue
		}
	}

	return nil
}
