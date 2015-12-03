package tester

import (
	"fmt"
	"log"

	"github.com/hashicorp/consul/api"
)

type verifier func() error

type Fuzz struct {
	Client  *api.Client
	Checks  []verifier
	Counter int
}

func NewFuzz(client *api.Client) (*Fuzz, error) {
	f := &Fuzz{Client: client}
	return f, nil
}

func (f *Fuzz) Populate() error {
	if _, err := f.fuzzRegister(); err != nil {
		return err
	}
	if _, err := f.fuzzSession(); err != nil {
		return err
	}
	if _, err := f.fuzzKV(); err != nil {
		return err
	}
	if _, err := f.fuzzACL(); err != nil {
		return err
	}
	return nil
}

func (f *Fuzz) Verify() error {
	log.Printf("Running %d fuzz checks...", len(f.Checks))
	for _, f := range f.Checks {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (f *Fuzz) generateName(base string) string {
	f.Counter++
	return fmt.Sprintf("%s%d", base, f.Counter)
}

func (f *Fuzz) fuzzRegister() (*api.CatalogRegistration, error) {
	reg := &api.CatalogRegistration{
		Node:    f.generateName("node"),
		Address: "127.0.0.1",
		Service: &api.AgentService{
			Service: f.generateName("service"),
			Port:    1234,
		},
		Check: &api.AgentCheck{
			Name:   f.generateName("check"),
			Status: "passing",
		},
	}

	catalog := f.Client.Catalog()
	if _, err := catalog.Register(reg, &api.WriteOptions{}); err != nil {
		return nil, err
	}

	f.Checks = append(f.Checks, func() error {
		node, _, err := catalog.Node(reg.Node, &api.QueryOptions{})
		if err != nil {
			return err
		}
		if node.Node.Node != reg.Node || node.Node.Address != reg.Address {
			return fmt.Errorf("bad: %v", node)
		}
		if _, ok := node.Services[reg.Service.Service]; !ok {
			return fmt.Errorf("bad: %v", node)
		}

		services, _, err := catalog.Service(reg.Service.Service, "", &api.QueryOptions{})
		if err != nil {
			return err
		}
		if len(services) != 1 {
			return fmt.Errorf("bad: %v", services)
		}
		service := services[0]
		if service.Node != reg.Node ||
			service.ServiceName != reg.Service.Service ||
			service.ServicePort != reg.Service.Port {
			return fmt.Errorf("bad: %v", service)
		}

		health := f.Client.Health()
		checks, _, err := health.Node(reg.Node, &api.QueryOptions{})
		if err != nil {
			return err
		}
		if len(checks) != 1 {
			return fmt.Errorf("bad: %v", checks)
		}
		check := checks[0]
		if check.Node != reg.Node ||
			check.Name != reg.Check.Name ||
			check.Status != reg.Check.Status {
			return fmt.Errorf("bad: %v", check)
		}
		return nil
	})

	return reg, nil
}

func (f *Fuzz) fuzzSession() (string, error) {
	reg, err := f.fuzzRegister()
	if err != nil {
		return "", err
	}

	s := &api.SessionEntry{
		Node:   reg.Node,
		Checks: []string{reg.Check.Name},
	}

	session := f.Client.Session()
	id, _, err := session.Create(s, &api.WriteOptions{})
	if err != nil {
		return "", err
	}

	f.Checks = append(f.Checks, func() error {
		entries, _, err := session.Node(reg.Node, &api.QueryOptions{})
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			return fmt.Errorf("bad: %v", entries)
		}
		entry := entries[0]
		if entry.ID != id || entry.Node != reg.Node {
			return fmt.Errorf("bad: %v", entry)
		}
		return nil
	})

	return id, nil
}

func (f *Fuzz) fuzzKV() (*api.KVPair, error) {
	p := &api.KVPair{
		Key:   f.generateName("key"),
		Value: []byte(f.generateName("value")),
	}

	kv := f.Client.KV()
	_, err := kv.Put(p, &api.WriteOptions{})
	if err != nil {
		return nil, err
	}

	f.Checks = append(f.Checks, func() error {
		pair, _, err := kv.Get(p.Key, &api.QueryOptions{})
		if err != nil {
			return err
		}
		if pair.Key != p.Key || string(pair.Value) != string(p.Value) {
			return fmt.Errorf("bad: %v", pair)
		}
		return nil
	})

	return p, nil
}

func (f *Fuzz) fuzzACL() (string, error) {
	a := &api.ACLEntry{
		Name: f.generateName("acl"),
		Type: "client",
	}

	acl := f.Client.ACL()
	id, _, err := acl.Create(a, &api.WriteOptions{Token: "root"})
	if err != nil {
		return "", err
	}

	f.Checks = append(f.Checks, func() error {
		entry, _, err := acl.Info(id, &api.QueryOptions{})
		if err != nil {
			return err
		}
		if entry.ID != id || entry.Name != a.Name {
			return fmt.Errorf("bad: %v", entry)
		}
		return nil
	})

	return id, nil
}
