package commands

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/cli"
)

func FillCommandFactory() (cli.Command, error) {
	return &Fill{}, nil
}

type Fill struct {
}

func (c *Fill) Help() string {
	helpText := `
Usage consul-live fill -keys=<n> -size=<bytes>
`
	return strings.TrimSpace(helpText)
}

func (c *Fill) Synopsis() string {
	return "Fills Consul's KV store"
}

func (c *Fill) Run(args []string) int {
	var keys int
	var size int
	cmdFlags := flag.NewFlagSet("fill", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.IntVar(&keys, "keys", 1024, "")
	cmdFlags.IntVar(&size, "size", 128, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if size < 1 {
		log.Println("Size must be at least one byte")
		return 1
	}

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Println(err)
		return 1
	}

	if err := c.run(client, keys, size); err != nil {
		log.Println(err)
		return 1
	}

	return 0
}

func (c *Fill) run(client *api.Client, keys int, size int) error {
	kv := client.KV()

	root, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	for i := 0; i < keys; i++ {
		buf := make([]byte, size)
		if _, err := rand.Read(buf); err != nil {
			return err
		}

		inner := fmt.Sprintf("%s/%d", root, i+1)
		if _, err := kv.Put(&api.KVPair{Key: inner, Value: buf}, nil); err != nil {
			return err
		}
	}
	return nil
}
