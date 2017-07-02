Nexus
==============

Nexus is a generic lambda engine with lots of awesome baked-in capabilities, such as CRON, structured data store, and a permissions/accounts system.


## Getting Started

### Download and build

*Please make sure you have [Go](https://golang.org/doc/install) 1.8 or greater installed.*

```shell
git clone https://github.com/twitchyliquid64/nexus
cd nexus
export GOPATH=`pwd`
go build nexus/serv
go build nexus/maintenance-tool
```

### Setup the first account

Both of these commands will prompt you for information.

```shell
./maintenance-tool createuser
./maintenance-tool resetauth
```

### Run the server!

Simple example:

`./serv --listener :8080`

More complicated, automatically fetching a LetsEncrypt certificate and running over HTTPS:

`./serv --key-cache /etc/letsencrypt.nexus --listener :443 --domains my-server-domain.com`