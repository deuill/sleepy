## Sleepy, the lightweight web application framework

### Introduction

Sleepy is a web application framework with a client - server architecture and a
lightweight approach. It attempts to provide the bare essentials for building a
web application, is opinionated but tries to keep its conventions at a minimum.

This repository contains the server part of Sleepy, written in Go, which runs as
a daemon and is in general never directly used. For more information on the Sleepy
client, written in PHP, check the corresponding [repository](https://github.com/deuill/sleepy-client).

Be aware that this is **alpha software**, and as such may crash your computer/eat
puppies/hide your socks. However, I have been running Sleepy on my own servers
for the past year, with uptime of months at a time, and have encountered no
serious issues. I still can't match a pair of socks, though.

### Building Sleepy

Assuming you already have all the dependancies required for building Sleepy via
```go get```, it's simply a matter of running ```make``` in the root directory. Installing
Sleepy requires running ```make install```, for installing the binaries, and ```make install-data```
for installing the configuration files and SQLite database. Alternatively, you can
run ```make package``` to build a package you can then redistribute.

For ArchLinux users, there exists a PKGBUILD file you can build directly in its
directory via ```makepkg```. This sets up Sleepy to run as a restricted *"http"* user
by default, which should be safer if things go south. It also installs a service
file for systemd under the name *"sleepy"*.

Init files exist for Debian init, SysV init (Fedora, CentOS etc.) and systemd.

### Running/Configuring Sleepy

Sleepy installs its configuration files in *"/etc/sleepy"*, in a common .ini format.
Most defaults are fine, though you will most likely need to change the address for
the http server, as well as the username/password used for your database.

Running Sleepy is simply a matter of running the ```sleepyd``` binary, installed in
*"/usr/bin"* by default, though running through an init file is probably better.

### Anything else?

Sleepy is not of much use alone, so you most likely want to set up the client
framework. For help, check the Sleepy client's [repository](https://github.com/deuill/sleepy-client).

### License

The server part is licensed under the MIT license. The client part is licensed
under seperate tems, check the corresponding repository for more information.
