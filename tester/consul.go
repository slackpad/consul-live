package tester

import (
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/consul/api"
)

type Consul struct {
	Command *exec.Cmd
}

func NewConsul(executable string, args []string) (*Consul, error) {
	c := &Consul{}
	c.Command = exec.Command(executable, args...)
	c.Command.Stdout = os.Stdout
	c.Command.Stderr = os.Stderr
	return c, nil
}

func (c *Consul) Start() error {
	if err := c.Command.Start(); err != nil {
		return err
	}
	return nil
}

func (c *Consul) Shutdown() error {
	if c.Command == nil {
		return nil
	}

	if err := c.Command.Process.Kill(); err != nil {
		return err
	}

	if _, err := c.Command.Process.Wait(); err != nil {
		return err
	}

	c.Command = nil
	return nil
}

func (c *Consul) WaitForLeader() error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	catalog := client.Catalog()
	for {
		_, meta, err := catalog.Nodes(&api.QueryOptions{})
		if err != nil {
			goto RETRY
		}
		if meta.KnownLeader && meta.LastIndex > 0 {
			return nil
		}

	RETRY:
		time.Sleep(2 * time.Second)
	}
}
