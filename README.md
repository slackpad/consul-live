# consul-live

This is a very basic set of tools to help perform live testing with Consul for
things that are difficult to unit test. It currently knows how to perform a
very basic upgrade test where it runs a server through a series of upgrades and
makes sure all the state store data makes it between the different versions.

