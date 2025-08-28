# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25 AS builder

WORKDIR /app
RUN wget https://github.com/swaggo/swag/releases/download/v1.16.4/swag_1.16.4_Linux_x86_64.tar.gz && \
    tar -xzf swag_1.16.4_Linux_x86_64.tar.gz && \
    mv swag /usr/local/bin/swag && \
    rm swag_1.16.4_Linux_x86_64.tar.gz

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Generate Swagger documentation
RUN ./generate-spec.sh

RUN CGO_ENABLED=0 GOOS=linux go build -o server
RUN CGO_ENABLED=0 GOOS=linux go build -C migrations -o migrate

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/migrations/migrate .
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

EXPOSE 8000

ENV GIN_MODE=release
CMD ["sh", "-c", "./migrate up head && ./server"]