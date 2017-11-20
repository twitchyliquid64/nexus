Nexus
==============

Nexus is a Lambda engine, Slack/IRC aggregator, configurable dashboard, and more for the meandering hacker/developer.

The lambda engine has lots of awesome baked-in capabilities:

 * APIs for
     * Making and recieving Web requests.
     * Reading/writing files from any associated storage buckets / system.
     * Saving/querying information to/from a structured data store.
     * Saving/retrieving from a key-value store.
     * Sending Emails.
     * Anything you can think of! Making new APIs is very easy.
 * To assist development / monitoring there is a full logging system, with automatic deletion after 28 days.
 * Lambda's are triggered by
     * A Schedule (cronspec)
     * HTTP request
     * Manually
 * Lambdas can be trivially and rapidly created in Javascript using the builtin editor, complete with autocomplete.
 * Logs and run explorer

Other features:

 * Message aggregator presents all your Slack channels and IRC channels in a single interface, staying connected persistently and saving all messages in it's database.
 * Can automatically fetch and manage SSL certificates using LetsEncrypt, and will automatically renew them as they expire.
 * Can automatically backup the database to S3 on a schedule.
 * Fully customizable dashboard
     * Supports cards, lists, graphs etc, all beautifully and extensively styled.
     * Dashboard can be updated by Lambda's via the file system.
 * A flexible authentication system supports password and OTP authentication, and any combination thereof.

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

## Wishlist / TODO

 - Ability to initiate DMs in the messenger.
