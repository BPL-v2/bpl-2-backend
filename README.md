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

## Running additional go tools

Install tools

```
make install-tools
```

make sure, the location where go install moves tools to is part of your path (on linux its ~/go/bin)

## Generating the openapi spec

### Annotating routes

To make sure that all api routes are properly documented, we use swag annotations as comments on all routes
Running `make swagger` will generate the files in the /docs directory which will be served under the route /api/swagger/doc.json

## DB migration

We use a custom migration tool to execute sql files for db migrations that uses files named "n-m.sql" to migrate from version n to version m.

To run this, run

```
make migrate-up
```

in the root directory to migrate up n versions or

```
make migrate-up head
```

to migrate to the latest version

## Run application

Make sure you've migrated the database to the latest version, added the .env file and run

```sh
make dev
```

## Creating a JWT for local testing

Some endpoints can only be called while authenticated via bearer token.
The login via PoE oauth only works on the production website, since redirect urls etc are verified by the oauth provider.
You can create your own token for local use however.

Run `make create-token ID=1 PERMISSIONS=admin,manager` with the user id and permissions you want to test and add the resulting JWT via devtools on your frontend running on localhost

F12-> application -> local storage:

- Key: auth
- Value: `YourJWT`
