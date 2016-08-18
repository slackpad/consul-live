package tester

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-uuid"
	"github.com/miekg/dns"
	"github.com/mitchellh/cli"
)

func LoadCommandFactory() (cli.Command, error) {
	return &Load{}, nil
}

type Load struct {
}

func (c *Load) Help() string {
	helpText := `
Usage consul-live load -actors=n -rate=rate
`
	return strings.TrimSpace(helpText)
}

func (c *Load) Synopsis() string {
	return "Loads the local Consul agent with realistic usage"
}

func (c *Load) Run(args []string) int {
	var actors int
	var rate int
	cmdFlags := flag.NewFlagSet("load", flag.ContinueOnError)
	cmdFlags.Usage = func() { log.Println(c.Help()) }
	cmdFlags.IntVar(&actors, "actors", 1, "")
	cmdFlags.IntVar(&rate, "rate", 10, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if actors < 1 {
		log.Println("At least one actor is required")
		return 1
	}
	if rate < 1 {
		log.Println("Rate must be at least 1 event/second")
		return 1
	}

	var wg sync.WaitGroup
	for i := 0; i < actors; i++ {
		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			log.Println("Could not make client: %s", err.Error())
			return 1
		}

		wg.Add(1)
		go func() {
			if err := load(client, rate); err != nil {
				log.Println(err.Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return 0
}

func opKeyCRUD(client *api.Client) error {
	kv := client.KV()

	root, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	_, err = kv.Put(&api.KVPair{Key: root}, nil)
	if err != nil {
		return err
	}

	inner := fmt.Sprintf("%s/inner", root)
	value := []byte("hello")
	_, err = kv.Put(&api.KVPair{Key: inner, Value: value}, nil)
	if err != nil {
		return err
	}

	pair, _, err := kv.Get(inner, nil)
	if err != nil {
		return err
	}
	if !bytes.Equal(pair.Value, value) {
		return fmt.Errorf("bad value: %#v", *pair)
	}

	value = []byte("world")
	_, err = kv.Put(&api.KVPair{Key: inner, Value: value}, nil)
	if err != nil {
		return err
	}

	pair, _, err = kv.Get(inner, nil)
	if err != nil {
		return err
	}
	if !bytes.Equal(pair.Value, value) {
		return fmt.Errorf("bad value: %#v", *pair)
	}

	_, err = kv.DeleteTree(root, nil)
	if err != nil {
		return err
	}

	return nil
}

func opGlobalLock(client *api.Client) error {
	opts := &api.LockOptions{
		Key:          "global",
		SessionTTL:   "10s",
		LockWaitTime: 20 * time.Second,
	}

	lock, err := client.LockOpts(opts)
	if err != nil {
		return err
	}

	if _, err := lock.Lock(nil); err != nil {
		return err
	}

	if err := lock.Unlock(); err != nil {
		return err
	}

	return nil
}

func opGlobalServiceRegister(client *api.Client) error {
	index := rand.Intn(128)
	service := &api.AgentServiceRegistration{
		ID:   fmt.Sprintf("fuzz-test:%d", index),
		Name: "fuzz-test",
		Port: 10000 + index,
	}

	agent := client.Agent()
	if err := agent.ServiceRegister(service); err != nil {
		return err
	}

	return nil
}

func opGlobalServiceDNSLookup(client *api.Client) error {
	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion("fuzz-test.service.consul.", dns.TypeSRV)
	if _, _, err := c.Exchange(m, "127.0.0.1:8600"); err != nil {
		return err
	}

	m.SetQuestion("fuzz-test.service.consul.", dns.TypeANY)
	if _, _, err := c.Exchange(m, "127.0.0.1:8600"); err != nil {
		return err
	}

	return nil
}

func load(client *api.Client, rate int) error {
	ops := []func(*api.Client) error{
		opKeyCRUD,
		opGlobalLock,
		opGlobalServiceRegister,
		opGlobalServiceDNSLookup,
	}

	minTimePerOp := time.Second / time.Duration(rate)
	for {
		start := time.Now()
		opIndex := rand.Intn(len(ops))
		if err := ops[opIndex](client); err != nil {
			return err
		}
		elapsed := time.Now().Sub(start)
		time.Sleep(minTimePerOp - elapsed)
	}

	return nil
}
