# BPL Backend

## How to set this up locally

Clone the infrastructure repo

```
git clone git@github.com:BPL-v2/bpl-2-infrastructure.git
cd bpl-2-infrastructure/local
docker compose up -d
```

This runs docker containers with nginx, postgres and kafka

Also copy /local/.env from this repo here.
For oauth / requests against poe api you'll need to also set some secret values

You can set up your own discord server/bot for testing by following the instructions over at https://github.com/BPL-v2/bpl-2-discord-bot

## Generating the openapi spec

### Annotating routes

To make sure that all api routes are properly documented, we use swag annotations as comments on all routes

### Installing swag

```
wget https://github.com/swaggo/swag/releases/download/v1.16.4/swag_1.16.4_Linux_x86_64.tar.gz -P tmp
tar -xzf tmp/swag_1.16.4_Linux_x86_64.tar.gz -C tmp
sudo mv tmp/swag /usr/local/bin/
rm -rf tmp
```

### Running it

```
swag init
./cleanup-swagger.sh
```

This will generate the files in the /docs directory which will be served under the route /api/swagger/doc.json

## DB migration

We use a custom migration tool to execute sql files for db migrations that uses files named "n-m.sql" to migrate from version n to version m.

To run this, run

```
./migrate [up|down] n
```

in the root directory to migrate up/down n versions or

```
./migrate up head
```

to migrate to the latest version

## Run application

```sh
go run main.go
```
