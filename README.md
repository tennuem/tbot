# tbot [![Build Status](https://github.com/tennuem/tbot/workflows/build/badge.svg)](https://github.com/tennuem/tbot/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/tennuem/tbot)](https://goreportcard.com/report/github.com/tennuem/tbot) [![Coverage Status](https://coveralls.io/repos/github/tennuem/tbot/badge.svg?branch=master)](https://coveralls.io/github/tennuem/tbot?branch=master)

## Configuration

| Command line          | Environment                | Default | Description     |
| --------------------- | :------------------------- | :------ | :-------------- |
| telegram.token        | TBOT_TELEGRAM_TOKEN        | string  | telegram token  |
| telegram.emoji_set    | TBOT_TELEGRAM_EMOJI_SET    | empty   | custom emoji set for service logos; empty disables logos |
| spotify.client_id     | TBOT_SPOTIFY_CLIENT_ID     | string  | telegram token  |
| spotify.client_secret | TBOT_SPOTIFY_CLIENT_SECRET | string  | telegram token  |
| yandex.proxy_url      | TBOT_YANDEX_PROXY_URL      | empty   | HTTP(S) proxy URL for Yandex Music only |
| mongodb.addr          | TBOT_MONGODB_ADDR          | string  | MongoDB URI     |
| sqlite.data_source    | TBOT_SQLITE_DATA_SOURCE    | ./tbot.db | SQLite file path; Compose uses /data/tbot.db |
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

OpenSpec is configured for Codex, OpenCode, and Oh My Pi. Install the pinned local
CLI with `make openspec-install`, refresh integrations with `make openspec-init`,
and check specifications with `make openspec-validate`. See
[the OpenSpec guide](docs/openspec.md) for the workflow and existing capability specs.

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

## Link cache and Telegram replies

Commands `/start`, `/help`, `/ping`, and `/list` work both in private chats and
groups, including `/help@bot_username` and commands followed by arguments.
Commands addressed to another bot are ignored. `/start` and `/help` show usage;
`/list` returns the sender's track history. `/ping` shows `pong`, the approximate
time since the command was sent (Telegram timestamps have one-second resolution),
and bot uptime. This is message age, not a full network round-trip measurement.
Telegram send failures are logged without exposing the bot token.

History can be exported to portable JSON and imported from bot archives or a
processed Telegram HTML export using `tbot history extract|import|export`.
Import defaults to preview mode and requires `--apply` to write data. See
[the history guide](docs/history.md) for extraction, sender attribution, backup,
Docker commands, and import/export examples. `/chatid` shows the current chat ID;
large `/list` responses are paginated with `/list 2`, `/list 3`, etc.

Incoming and discovered links are normalized before lookup and storage. Sharing
parameters (`utm_*`, `ref_id`, `si`, advertising click IDs) and fragments are removed;
track identifiers such as YouTube's `v` and Apple Music's `i` are retained.
Yandex regional domains, Spotify locale prefixes, and YouTube short links are
normalized to the provider's canonical host.

SQLite stores the complete set of provider links for each track and indexes aliases
by provider and URL. Sending a discovered link from another service therefore finds
the same cached answer. A cache hit returns immediately and schedules a background
refresh, with one refresh per track and at most four concurrent refreshes. If all
slots are occupied, the cached answer is still returned and another request can retry
the refresh. Temporary provider failures retain previously found links; a definitive
not-found response removes that provider's link from the next answer.

Replies use Telegram's standard HTML formatting with clickable service names,
without an inline keyboard or web page previews. Cached replies can show the first
sender and send time (Moscow time), only in the original chat. Sender information
from another chat is never included in the reply.

### Service logos

The [TbotMusicServices emoji set](https://t.me/addemoji/TbotMusicServices) contains
the four service logos. To enable it in your server's Compose file, add this to the
bot's existing `environment` section (no env file is required):

```yaml
TBOT_TELEGRAM_EMOJI_SET: TbotMusicServices
```

With this repository's Compose file, you can instead run:

```fish
env TBOT_TELEGRAM_EMOJI_SET=TbotMusicServices docker compose up -d --build app
```

The setting accepts the set's short name, not its full URL. At startup, the bot
uses its existing Telegram token to read `getStickerSet` and matches icons by their
assigned emoji: Spotify 🎧, Yandex Music 🎵, Apple Music 🎶, YouTube Music ▶️.
The order of the icons in the set does not matter. Logos appear beside the HTML
links using `tg-emoji`; the existing Telegram library needs no upgrade.

Loading has a 10-second timeout. If loading fails, the bot logs a warning and uses
text links; an absent individual logo also falls back to a text link. An empty
setting disables logos. The set is read once per process start; changing Compose
environment values requires recreating the container. For sending custom emoji in
private chats and groups, the bot's owner must have Telegram Premium or the bot
must have an additional username purchased on Fragment; see
[Telegram's requirements](https://core.telegram.org/bots/api#html-style).

On first startup, existing SQLite rows are imported into new cache tables in a
transaction. The old `links` table is retained. The previous schema did not store
complete link sets or send dates: imported records are rebuilt on their next use,
and unknown historical dates are left unknown. Keep the SQLite database on a
persistent Docker volume and back it up before updating the application.

### Keeping SQLite across container recreation

The running bot uses SQLite, even if MongoDB is also present in Compose. This
repository's Compose file mounts `./db` from the host at `/data` and sets
`TBOT_SQLITE_DATA_SOURCE=/data/tbot.db`. The cache is therefore kept in
`./db/tbot.db` on the host and survives `--force-recreate`. The `db` directory is
excluded from Git and the Docker build context. Keep the same host directory when
deploying future versions. For a separate server Compose file, add to `app`:

```yaml
environment:
  TBOT_SQLITE_DATA_SOURCE: /data/tbot.db
volumes:
  - ./db:/data
```

Merge these entries into the existing `environment` and `volumes` sections.
Use `TBOT_SQLITE_DATA_SOURCE`, not `TBOT_DATA_SOURCE`: the latter is not read by
the application and leaves the default relative `./tbot.db` path in use. Mount
the database directory rather than an individual file; Docker can otherwise
create a directory at a missing bind source that was intended to be a file.
Paths such as `./db` are relative to the Compose project directory.
`docker compose restart app` keeps a container's writable layer, whereas
`--force-recreate` replaces it. Files stored only in that layer are not preserved.

For an existing container using the old default `/tbot.db`, copy its database
**before changing Compose or recreating the container**. If an explicit SQLite
path was configured, use that source path instead. Do not overwrite an existing
host database; back it up first.

```fish
mkdir -p db
docker compose stop app
docker compose cp app:/tbot.db ./db/tbot.db
```

Only after the copy succeeds, add the configuration above and recreate `app`:

```fish
docker compose up -d --no-deps --force-recreate app
```

Mounting a directory later does not recover a database from a container that has
already been deleted. To check reuse, share the same link in the same chat and
look for `cache=hit` in the logs. First-sender metadata is omitted in another chat;
old imported rows also have no original chat ID or timestamp. The absence of that
caption alone does not mean the cache is empty.

Logs distinguish `cache=miss`, `cache=hit`, and `cache=error`, and report background
refreshes and provider failures separately. Normal cache misses and ordinary chat
messages are not logged as application errors. Startup no longer prints credentials.
Spotify uses a synchronized OAuth token source instead of a token-refresh goroutine,
so concurrent foreground searches and background refreshes share tokens safely.
