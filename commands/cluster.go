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
Usage consul-live cluster -servers=<int> -server-args=<string> -clients=<int> -client-args=<string>
`
	return strings.TrimSpace(helpText)
}

func (c *Cluster) Synopsis() string {
	return "Starts up a cluster with the given parameters"
}

func (c *Cluster) Run(args []string) int {
	cfg := &live.ClusterConfig{
		Executable: "consul",
	}
	cmdFlags := flag.NewFlagSet("cluster", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.IntVar(&cfg.Servers, "servers", 3, "")
	cmdFlags.Var(&stringsFlag{&cfg.ServerArgs}, "server-args", "")
	cmdFlags.IntVar(&cfg.Clients, "clients", 10, "")
	cmdFlags.Var(&stringsFlag{&cfg.ClientArgs}, "client-args", "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	log.Println(cfg)

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
