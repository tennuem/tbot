# History import and export

History tools are operator commands in the same `tbot` binary. They do not start
Telegram polling, send messages, or query music providers. Use an image built
with these tools; verify that `tbot history import --help` lists `--input` and
`--apply` before using the Docker commands below.

## Extract a Telegram HTML export

```fish
./bin/tbot history extract \
  --input /path/to/ChatExport \
  --output ./history.json \
  --csv ./history.csv \
  --timezone Europe/Moscow \
  --chat-id=-1001234567890 \
  --exclude-users 123456789
```

`--input` accepts a single HTML file or a directory containing `messages.html`,
`messages2.html`, and subsequent pages. The original files are never modified.
Sender names, numeric IDs from avatar filenames, message IDs, dates, explicit
links, plain text links, and primary link previews are extracted. Joined messages
inherit their sender, including across pages. Missing numeric IDs remain unknown;
display names are never used to guess another account's ID.
Links in preview descriptions and quoted replies are not attributed to the sender.

Set `--exclude-users` to the bot's numeric user ID, or a comma-separated list of
IDs, to avoid importing its generated replies as human submissions. Deleted
accounts retain their exported display name and `user_id: 0`; an absent ID is
never guessed. Missing names, malformed dates, or invalid URLs stop extraction
unless `--allow-incomplete` is explicitly supplied. A JSON report counts these
conditions and lists authors and providers.

The HTML timezone is explicit because exported timestamps have no UTC offset.
For group links `https://t.me/c/1234567890/42`, the Bot API chat ID is
`-1001234567890`; `/chatid` can also show the ID in the bot. If `--chat-id` is
omitted during extraction, bind the unassigned shares during import with that
flag. Already assigned chats are never moved by this flag.

All valid HTTP(S) links are retained in the archive and CSV, including non-music
links. Only supported track-shaped URLs are indexed in the music cache. Tracking
parameters are removed. Repeated submissions remain separate history events;
duplicates of a text link and its preview in one message are counted once.

## Preview and apply an import

The portable format is versioned JSON. Import accepts this bot format, including
files created by `history extract` and `history export`.

```fish
./bin/tbot history import --input ./history.json --sqlite ./db/tbot.db
```

Without `--apply`, this only validates the archive and prints counts: SQLite is
not opened, created, or migrated. Review the report before applying it.

For this repository's Compose file, place the archive at `./db/history.json`:

```fish
docker compose exec -T app /bin/tbot history import --input /data/history.json --sqlite /data/tbot.db
```

Stop the application and back up its database before applying an import. The
following paths match this repository's `./db:/data` mount; adapt them to a custom
server Compose file. For example, a container path `/db/tbot.db` requires
`--sqlite /db/tbot.db` and an archive accessible in that same mounted directory.
The environment variable is **`TBOT_SQLITE_DATA_SOURCE`**, not `TBOT_DATA_SOURCE`.

```fish
docker compose stop app
cp ./db/tbot.db ./db/tbot-before-history-(date +%Y%m%d-%H%M%S).db
docker compose run --rm --no-deps app history import --input /data/history.json --sqlite /data/tbot.db --apply
docker compose start app
```

Only continue to the next step if the preceding command succeeds. Applying an
import merges cache entries, historical aliases, and submissions in a single data
transaction. Reimporting the same archive does not duplicate its submissions.
Earlier imported sender metadata is retained without overwriting a more recently
checked title or provider links. Previously known submissions are preserved when
an earlier sender is discovered. No chat messages are replayed.

Extracted track entries are intentionally incomplete cache records. Music
providers are checked on their next use; this avoids treating old preview titles
or inferred cross-service matches as verified current data. The original sender
and date can already be shown on that first use in the matching chat.

New submissions are recorded independently of the cache, including cache hits.
`/list` includes tracks a person shared even when someone else shared them first.
It shows the latest 100 distinct tracks, with `/list 2`, `/list 3`, etc. for pages
that fit Telegram's message limit. Full exports have no 100-track history limit.

## Export bot history

```fish
./bin/tbot history export --sqlite ./db/tbot.db --output ./history-backup.json --csv ./history-backup.csv
```

Or, with this repository's running Compose service:

```fish
docker compose exec -T app /bin/tbot history export --sqlite /data/tbot.db --output /data/history-backup.json
```

Export opens an existing database read-only and takes a consistent snapshot. It
includes sharing events, full cache records, and historical URL aliases. Old
cache-only records are included even when their complete submission history was
never recorded. Older messages not present in the cache or an imported archive
cannot be reconstructed.

Output files are created with mode `0600` and never overwrite existing files;
choose new filenames for subsequent exports. JSON is the import format; CSV is
for inspection and includes sender, timestamp, chat, message, provider, and URL.
