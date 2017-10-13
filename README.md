# consul-live

This is a very basic set of tools to help perform live testing with Consul.

```
usage: consul-live [--version] [--help] <command> [<args>]

Available commands are:
    cluster    Starts up a cluster with the given parameters
    kill       Kills the current leader once the cluster is stable
    load       Loads the local Consul agent with realistic usage
    upgrade    Runs Consul through a given series of in-place upgrades
```
