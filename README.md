# consul-live

This is a very basic set of tools to help perform live testing with Consul for
things that are difficult to unit test. It currently knows how to perform a
very basic upgrade test where it runs a server through a series of upgrades and
makes sure all the state store data makes it between the different versions.

```
usage: consul-live [--version] [--help] <command> [<args>]

Available commands are:
    upgrade    Usage consul-live upgrade base version1 ... versionN

  Starts Consul using the base executable then shuts it down and upgrades in
  place using the supplied version executables. The base version is populated
  with some test data and that data is verified after each upgrade.
```
