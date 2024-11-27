# BPL Backend

## How to set this up locally

Create .env file for environment variables and ask me very nicely for actually secret values

```
echo "DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=postgres

DISCORD_CLIENT_ID=dummy
DISCORD_CLIENT_SECRET=dummy

JWT_SECRET=dummy
" > .env
```

Start postgres database from docker-compose.yml

```sh
docker compose up -d
```

Run application

```sh
go run main.go
```
