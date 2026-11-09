# tbot [![Build Status](https://github.com/tennuem/tbot/workflows/build/badge.svg)](https://github.com/tennuem/tbot/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/tennuem/tbot)](https://goreportcard.com/report/github.com/tennuem/tbot) [![Coverage Status](https://coveralls.io/repos/github/tennuem/tbot/badge.svg?branch=master)](https://coveralls.io/github/tennuem/tbot?branch=master)

## Configuration

| Command line          | Environment                | Default | Description     |
| --------------------- | :------------------------- | :------ | :-------------- |
| telegram.token        | TBOT_TELEGRAM_TOKEN        | string  | telegram token  |
| spotify.client_id     | TBOT_SPOTIFY_CLIENT_ID     | string  | telegram token  |
| spotify.client_secret | TBOT_SPOTIFY_CLIENT_SECRET | string  | telegram token  |
| yandex.proxy_url      | TBOT_YANDEX_PROXY_URL      | empty   | HTTP(S) proxy URL for Yandex Music only |
| mongodb.addr          | TBOT_MONGODB_ADDR          | string  | MongoDB URI     |
| logger.level          | TBOT_LOGGER_LEVEL          | info    | level of logger |

## Usage

```fish
docker compose build
docker compose up -d
```

## Yandex Music proxy

Use a Russian proxy for Yandex Music while keeping the bot in Europe:

```fish
env TBOT_YANDEX_PROXY_URL=http://198.51.100.20:3128 docker compose up -d --build app
```

Replace the example IP with your Russian server's address. The setting also accepts
`https://proxy:port` and optional URL-encoded credentials (`http://user:password@proxy:port`).
It can be set with `--yandex.proxy_url` when running the binary. An explicit proxy
overrides environment proxy settings, including `NO_PROXY`, for Yandex requests.
If empty, Yandex uses Go's standard `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` behavior.
Other providers retain their existing settings. Requests time out after 10 seconds;
an unavailable dedicated proxy returns an error without a direct retry.
Invalid proxy URLs stop startup. The proxy URL is omitted from printed configuration.

See [the Russian proxy setup guide](docs/yandex-proxy.md) for Squid configuration,
access restrictions, checks, and rollback.

## Development

Requires Go 1.27.3 with CGO enabled (SQLite), a C compiler, and golangci-lint 2.15.0.
CI reads the Go version from `go.mod` and uses the v2 configuration in `.golangci.yml`.

To install the pinned linter into the project's ignored `bin` directory:

```fish
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v2.15.0/install.sh | sh -s -- -b ./bin v2.15.0
```

```fish
make build  # builds bin/tbot
make test   # full test suite with the race detector
make lint   # gofmt, govet, ineffassign, staticcheck
```

If the linter is installed in `bin` rather than on `PATH`, use
`make lint GOLANGCI_LINT=./bin/golangci-lint`.

The Docker build keeps CGO enabled and uses Go 1.27.3 on Alpine 3.24, with Alpine
3.24.4 in the runtime image. Module downloads and compilation use BuildKit caches;
the final image contains CA certificates and the stripped binary. `.dockerignore`
excludes local databases, credentials, Git metadata, and build artifacts.
