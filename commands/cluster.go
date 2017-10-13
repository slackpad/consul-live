package commands

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/slackpad/consul-live/live"
)

func ClusterCommandFactory() (cli.Command, error) {
	return &Cluster{}, nil
}

type Cluster struct {
}

func (c *Cluster) Help() string {
	helpText := `
Usage consul-live cluster <options>

Options:

-consul=<string>       Consul executable, defaults to "consul" from PATH
-servers=<int>         Number of servers, defaults to 3
-server-args=<string>  Additional args to pass to servers, may be given multiple times
-clients=<int>         Number of clients, defaults to 10
-client-args=<string>  Additional args to pass to clients, may be given multiple times
-nice-ports=<bool>     If true, uses the Consul default ports for the first agent, defaults to true
`
	return strings.TrimSpace(helpText)
}

func (c *Cluster) Synopsis() string {
	return "Starts up a cluster"
}

func (c *Cluster) Run(args []string) int {
	cfg := &live.ClusterConfig{}
	cmdFlags := flag.NewFlagSet("cluster", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.StringVar(&cfg.Executable, "consul", "consul", "")
	cmdFlags.IntVar(&cfg.Servers, "servers", 3, "")
	cmdFlags.Var(&stringsFlag{&cfg.ServerArgs}, "server-args", "")
	cmdFlags.IntVar(&cfg.Clients, "clients", 10, "")
	cmdFlags.Var(&stringsFlag{&cfg.ClientArgs}, "client-args", "")
	cmdFlags.BoolVar(&cfg.NicePorts, "nice-ports", true, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := c.run(cfg); err != nil {
		log.Println(err)
		return 1
	}

	return 0
}

func (c *Cluster) run(cfg *live.ClusterConfig) error {
	cluster, err := live.NewCluster(cfg)
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

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)
	<-wait
	log.Println("Got interrupt, cleaning up...")
	return nil
}
