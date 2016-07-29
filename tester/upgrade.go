package tester

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

func UpgradeCommandFactory() (cli.Command, error) {
	return &Upgrade{}, nil
}

type Upgrade struct {
}

func (c *Upgrade) Help() string {
	helpText := `
Usage consul-live upgrade base version1 ... versionN

  Starts Consul using the base executable then shuts it down and upgrades in
  place using the supplied version executables. The base version is populated
  with some test data and that data is verified after each upgrade.
`
	return strings.TrimSpace(helpText)
}

func (c *Upgrade) Synopsis() string {
	return "Runs Consul through a given series of in-place upgrades"
}

func (c *Upgrade) Run(args []string) int {
	if len(args) < 2 {
		log.Println("At least two versions must be given")
		return 1
	}

	if err := c.upgrade(args); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

type ServerConfig struct {
	Server           bool   `json:"server,omitempty"`
	Bootstrap        bool   `json:"bootstrap,omitempty"`
	Bind             string `json:"bind_addr,omitempty"`
	DataDir          string `json:"data_dir,omitempty"`
	Datacenter       string `json:"datacenter,omitempty"`
	ACLMasterToken   string `json:"acl_master_token,omitempty"`
	ACLDatacenter    string `json:"acl_datacenter,omitempty"`
	ACLDefaultPolicy string `json:"acl_default_policy,omitempty"`
	LogLevel         string `json:"log_level,omitempty"`
}

func (c *Upgrade) upgrade(versions []string) error {
	base := versions[0]
	versions = versions[1:]

	var dir string
	var err error
	dir, err = ioutil.TempDir("", "consul")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	config, err := ioutil.TempFile(dir, "config")
	if err != nil {
		return err
	}

	content, err := json.Marshal(ServerConfig{
		Server:           true,
		Bootstrap:        true,
		Bind:             "127.0.0.1",
		DataDir:          dir,
		Datacenter:       "dc1",
		ACLMasterToken:   "root",
		ACLDatacenter:    "dc1",
		ACLDefaultPolicy: "allow",
	})
	if err != nil {
		return err
	}
	if _, err := config.Write(content); err != nil {
		return err
	}
	if err := config.Close(); err != nil {
		return err
	}

	// Start the first version of Consul, which is our base.
	log.Printf("Starting base Consul from '%s'...\n", base)
	args := []string{
		"agent",
		"-config-file",
		config.Name(),
	}
	consul, err := NewConsul(base, args)
	if err != nil {
		return err
	}
	if err := consul.Start(); err != nil {
		return err
	}
	defer func() {
		if err := consul.Shutdown(); err != nil {
			log.Println(err)
		}
	}()

	// Wait for it to start up and elect itself.
	if err := consul.WaitForLeader(); err != nil {
		return err
	}

	// Populate it with some realistic data, enough to kick out a snapshot.
	log.Println("Populating with initial state store data...")
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}
	fuzz, err := NewFuzz(client)
	if err != nil {
		return err
	}
	for {
		if err := fuzz.Populate(); err != nil {
			return err
		}

		entries, err := ioutil.ReadDir(dir + "/raft/snapshots/")
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			break
		}
	}

	// Push some data in post-snapshot to make sure there's some stuff
	// in the Raft log as well.
	if err := fuzz.Populate(); err != nil {
		return err
	}
	if err := fuzz.Verify(); err != nil {
		return err
	}

	// Now shutdown the base version and try upgrading through the given
	// versions.
	if err := consul.Shutdown(); err != nil {
		return err
	}
	for _, version := range versions {
		// Start the upgraded version with the same data-dir.
		log.Printf("Upgrading to Consul from '%s'...\n", version)
		upgrade, err := NewConsul(version, args)
		if err != nil {
			return err
		}
		if err := upgrade.Start(); err != nil {
			return err
		}
		defer func() {
			if err := upgrade.Shutdown(); err != nil {
				log.Println(err)
			}
		}()

		// Wait for it to start up and elect itself.
		if err := upgrade.WaitForLeader(); err != nil {
			return err
		}

		// Make sure the data is still present.
		if err := fuzz.Verify(); err != nil {
			return err
		}

		// Add some new data for this version of Consul.
		if err := fuzz.Populate(); err != nil {
			return err
		}
		if err := fuzz.Verify(); err != nil {
			return err
		}

		// Shut it down in anticipation of the next upgrade.
		if err := upgrade.Shutdown(); err != nil {
			return err
		}
	}

	log.Println("Upgrade series complete")
	return nil
}
