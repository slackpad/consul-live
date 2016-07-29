package tester

import (
	"log"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

func LoadCommandFactory() (cli.Command, error) {
	return &Load{}, nil
}

type Load struct {
}

func (c *Load) Help() string {
	helpText := `
Usage consul-live load <type>

  Generates load against the local Consul agent. The following types
  are supported:

  kv - Perform random reads, writes, and locks against a set of KVs
  dns - Perform random DNS lookups against Consul (assuming port 8600)
`
	return strings.TrimSpace(helpText)
}

func (c *Load) Synopsis() string {
	return "Loads the local Consul agent with realistic usage"
}

func (c *Load) Run(args []string) int {
	if len(args) != 1 {
		log.Println("A single load type must be given")
		return 1
	}

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Println("Could not make client: %s", err.Error())
		return 1
	}

	loadType := args[0]
	switch loadType {
	case "kv":
		err = c.loadKV(client)

	case "dns":
		err = c.loadDNS(client)

	default:
		log.Println("Unsupported load type %q", loadType)
		return 1
	}
	if err != nil {
		log.Println(err.Error())
		return 1
	}

	return 0
}

func (c *Load) loadKV(client *api.Client) error {
	return nil
}

func (c *Load) loadDNS(client *api.Client) error {
	return nil
}
