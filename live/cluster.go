package live

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/consul/test/porter"
)

type ClusterConfig struct {
	Executable string
	NicePorts  bool
	Servers    int
	ServerArgs []string
	Clients    int
	ClientArgs []string
}

type Cluster struct {
	DataDir string
	Agents  []*Consul
}

func NewCluster(cfg *ClusterConfig) (*Cluster, error) {
	n := cfg.Servers + cfg.Clients
	if n < 1 {
		return nil, fmt.Errorf("at least one client or server required")
	}

	ports, err := porter.RandomPorts(5 * n)
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "cluster")
	if err != nil {
		return nil, err
	}
	var disarm bool
	defer func() {
		if !disarm {
			os.RemoveAll(dir)
		}
	}()

	var joinPort int
	baseArgs := func(idx int) []string {
		dnsPort := ports[5*idx+0]
		httpPort := ports[5*idx+1]
		lanPort := ports[5*idx+2]
		wanPort := ports[5*idx+3]
		serverPort := ports[5*idx+4]

		// Set the default ports on the first agent for convenience.
		if idx == 0 {
			if cfg.NicePorts {
				dnsPort = 8600
				httpPort = 8500
				lanPort = 8301
				wanPort = 8302
				serverPort = 8300
			}
			joinPort = lanPort
		}

		node := fmt.Sprintf("node-%d", lanPort)
		args := []string{
			"agent",
			"-node", node,
			"-data-dir", fmt.Sprintf("%s/%s", dir, node),
			"-retry-join", fmt.Sprintf("127.0.0.1:%d", joinPort),
			"-bind", "127.0.0.1",
			"-client", "127.0.0.1",
			"-hcl", fmt.Sprintf("ports={dns=%d http=%d serf_lan=%d serf_wan=%d server=%d}",
				dnsPort, httpPort, lanPort, wanPort, serverPort),
		}
		return args
	}

	var agents []*Consul
	for i := 0; i < cfg.Servers; i++ {
		args := append(baseArgs(i), []string{
			"-server",
			fmt.Sprintf("-bootstrap-expect=%d", cfg.Servers),
			"-hcl", "performance={raft_multiplier=1}",
		}...)
		args = append(args, cfg.ServerArgs...)
		consul, err := NewConsul(cfg.Executable, args)
		if err != nil {
			return nil, err
		}
		agents = append(agents, consul)
	}
	for i := 0; i < cfg.Clients; i++ {
		args := append(baseArgs(cfg.Servers+i), cfg.ClientArgs...)
		consul, err := NewConsul(cfg.Executable, args)
		if err != nil {
			return nil, err
		}
		agents = append(agents, consul)
	}

	disarm = true
	return &Cluster{
		DataDir: dir,
		Agents:  agents,
	}, nil
}

func (c *Cluster) Start() error {
	for i, consul := range c.Agents {
		if err := consul.Start(); err != nil {
			return err
		}

		// Sleep a bit so the later agents will have something to join
		// to without a backoff.
		if i == 0 {
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}

func (c *Cluster) Shutdown() error {
	for _, consul := range c.Agents {
		if err := consul.Shutdown(); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(c.DataDir); err != nil {
		return err
	}
	return nil
}
