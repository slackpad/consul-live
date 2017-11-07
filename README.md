# consul-live

This is a very basic set of tools to help perform live testing with Consul.

```
usage: consul-live [--version] [--help] <command> [<args>]

Available commands are:
    block         Runs a blocking queries against a cluster
    cluster       Starts up a cluster
    federation    Starts up a federation of clusters
    kill          Kills the current leader once the cluster is stable
    load          Loads the local Consul agent with realistic usage
    upgrade       Runs Consul through a given series of in-place upgrades
```
