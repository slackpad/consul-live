package commands

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/slackpad/consul-live/live"
)

func FederationCommandFactory() (cli.Command, error) {
	return &Federation{}, nil
}

type Federation struct {
}

func (c *Federation) Help() string {
	helpText := `
Usage consul-live federation <options>

Options:

-consul=<string>       Consul executable, defaults to "consul" from PATH
-datacenters=<int>     Number of datacenters, defaults to 3
-servers=<int>         Number of servers in each datacenter, defaults to 3
-server-args=<string>  Additional args to pass to servers, may be given multiple times
-clients=<int>         Number of clients in each datacenter, defaults to 3
-client-args=<string>  Additional args to pass to clients, may be given multiple times
-nice-ports=<bool>     If true, uses the Consul default ports for the first agent, defaults to true
`
	return strings.TrimSpace(helpText)
}

func (c *Federation) Synopsis() string {
	return "Starts up a federation of clusters"
}

func (c *Federation) Run(args []string) int {
	var dcs int
	cfg := &live.ClusterConfig{}
	cmdFlags := flag.NewFlagSet("federation", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.StringVar(&cfg.Executable, "consul", "consul", "")
	cmdFlags.IntVar(&dcs, "datacenters", 3, "")
	cmdFlags.IntVar(&cfg.Servers, "servers", 3, "")
	cmdFlags.Var(&stringsFlag{&cfg.ServerArgs}, "server-args", "")
	cmdFlags.IntVar(&cfg.Clients, "clients", 3, "")
	cmdFlags.Var(&stringsFlag{&cfg.ClientArgs}, "client-args", "")
	cmdFlags.BoolVar(&cfg.NicePorts, "nice-ports", true, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := c.run(dcs, cfg); err != nil {
		log.Println(err)
		return 1
	}

	return 0
}

func (c *Federation) run(dcs int, cfg *live.ClusterConfig) error {
	if dcs < 1 {
		return fmt.Errorf("At least one datacenter is required")
	}

	var wanJoin string
	for i := 0; i < dcs; i++ {
		dc := fmt.Sprintf("dc%d", i+1)
		cc := *cfg
		cc.ServerArgs = append(cc.ServerArgs, "-datacenter", dc)
		cc.ClientArgs = append(cc.ClientArgs, "-datacenter", dc)
		if i > 0 {
			cc.NicePorts = false
		}

		cluster, err := live.NewCluster(&cc)
		if err != nil {
			return err
		}
		defer func() {
			if err := cluster.Shutdown(); err != nil {
				log.Println(err)
			}
		}()
		if err := cluster.Start(); err != nil {
			return err
		}

		if i > 0 {
			agent := cluster.Client.Agent()
			if err := agent.Join(wanJoin, true); err != nil {
				return err
			}
		} else {
			wanJoin = cluster.WANJoin
		}
	}

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)
	<-wait
	log.Println("Got interrupt, cleaning up...")
	return nil
}
