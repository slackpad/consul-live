package commands

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
	"golang.org/x/net/http2"
)

func BlockCommandFactory() (cli.Command, error) {
	return &Block{}, nil
}

type Block struct {
}

func (c *Block) Help() string {
	helpText := `
Usage consul-live block <options>

Options:

-queries=<int>         Number of blocking queries, defaults to 1
`
	return strings.TrimSpace(helpText)
}

func (c *Block) Synopsis() string {
	return "Runs a blocking queries against a cluster"
}

func (c *Block) Run(args []string) int {
	var queries int
	cmdFlags := flag.NewFlagSet("block", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.IntVar(&queries, "queries", 1, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := c.run(queries); err != nil {
		log.Println(err)
		return 1
	}

	return 0
}

func (c *Block) run(queries int) error {
	cfg := api.DefaultConfig()
	tlsccfg, err := api.SetupTLSConfig(&cfg.TLSConfig)
	if err != nil {
		return err
	}

	transport := cfg.Transport
	transport.TLSClientConfig = tlsccfg
	if err := http2.ConfigureTransport(transport); err != nil {
		return err
	}
	cfg.HttpClient = &http.Client{
		Transport: transport,
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	kv := client.KV()
	for i := 0; i < queries; i++ {
		path := fmt.Sprintf("block/%d", i+1)
		if _, err := kv.Put(&api.KVPair{Key: path}, nil); err != nil {
			return err
		}

		go func(key string) {
			qo := &api.QueryOptions{}
			for {
				_, qm, err := kv.Get(key, qo)
				if err != nil {
					panic(err)
				}
				if qo.WaitIndex > 0 {
					log.Printf("%s woke up", path)
				}
				qo.WaitIndex = qm.LastIndex
			}
		}(path)
	}

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)
	<-wait
	return nil
}
