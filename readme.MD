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

## Wishlist

 - Websockets to get logs in realtime from the run-explorer view without refreshing.
 - Ability to initiate DMs in the messenger.
 - Ability to sort conversations by last-recieved-message in the messenger.
 - Ability to insert/fetch/query API for datastores from integrations.
 - Support S3 / Drive as filesystem sources.
 - Have a Settings UI to setup things like messenger sources + filesystem sources.
