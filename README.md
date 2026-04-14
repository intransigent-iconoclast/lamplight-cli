# lamplight-cli

A CLI for finding and downloading books. Talks to your self-hosted Prowlarr or Jackett instance, searches across all your configured indexers, and pipes results straight to Deluge. That's it.

Built in Go. Open source.

---

## what it does

- Search for books across multiple torrent indexers at once
- Filter results by format — epub, pdf, mobi, audiobook, comic, or just `book` to catch all prose formats
- Sort by seeders, leechers, size, or title
- Download straight to Deluge with one command
- Tracks download history and syncs status from Deluge
- Organizes completed downloads into your library automatically — reads embedded metadata from epub, mp3, and m4b files
- Handles multi-file downloads (audiobook chapters, comic bundles) as a group
- Syncs indexers from Prowlarr/Jackett automatically, filtering to only book-supporting ones

---

## commands

### search
```bash
lamplight search "consider phlebas"
lamplight search "dune" -l 40 -t epub
lamplight search "one piece" -t comic -s seeders
lamplight search "stephen king" -t book,audiobook -l 100
```

| flag | what it does |
|------|-------------|
| `-l` | how many results to fetch (default 15, use 0 for no limit) |
| `-t` | filter by type: `book`, `epub`, `pdf`, `mobi`, `audiobook`, `comic`, `unknown`, `all` — comma-separated works |
| `-s` | sort by: `seeders` (default), `leechers`, `size`, `title` |
| `-b` | book category filter, on by default |
| `-i` | search a specific indexer by index |

Results are cached for 30 minutes. After that you'll need to re-run your search.

### download
```bash
lamplight download 3            # downloads result #3 from your last search
lamplight download 3 --force    # re-download even if it's already in history, or search results are stale
```

lamplight blocks duplicate downloads — if that link is already in your history it'll tell you. use `--force` to download anyway.

### history
```bash
lamplight history list
lamplight history list --filter failed
lamplight history list "dune"                     # search by title (substring)
lamplight history list "herbert" --filter completed

lamplight history sync                            # poll deluge for status updates
lamplight history retry 3                         # re-send entry #3 to deluge
lamplight history retry --all-failed              # re-send everything that failed at once
lamplight history update 3 --status failed        # manually fix a stuck entry
lamplight history clear
```

the index shown in `history list` is always the global index — use it directly with `retry` or `update` even when filtering.

### organize
```bash
lamplight organize                                # process all completed downloads
lamplight organize ~/Downloads/some-book.epub     # one-off manual file or folder
lamplight organize --dry-run                      # preview without moving anything
```

run `history sync` first to make sure statuses are up to date, then `organize` to move everything into your library.

files with complete metadata (author + title) go into:
```
<library-path>/library/<author>/<title> (<year>).<ext>
```

everything else ends up in:
```
<library-path>/uncategorized/<filename>
```

**multi-file downloads** (audiobook chapters, comic bundles) are kept together in a folder:
```
<library-path>/library/<author>/<title> (<year>)/01 - Chapter One.mp3
                                                  02 - Chapter Two.mp3
```

if you have both an ebook and an audiobook of the same title, they sit side by side under the same author folder without conflicting:
```
library/
  Frank Herbert/
    Dune (1965).epub
    Dune (1965)/
      01.mp3
      02.mp3
```

if you set `--audiobook-path`, audiobooks go to a completely separate root:
```
/mnt/media/books/library/Frank Herbert/Dune (1965).epub
/mnt/media/audiobooks/library/Frank Herbert/Dune (1965)/01.mp3
```

if a filename already exists, lamplight appends `_2`, `_3`, etc rather than overwriting.

**metadata sources**, in order of preference:
- epub: Dublin Core fields from the OPF file inside the zip
- mp3: ID3v2 tags (supports UTF-8, UTF-16 LE/BE)
- m4b/m4a: iTunes MP4 atoms (`©nam`, `©ART`, `©day`)
- fallback: `Author - Title` filename pattern

### config
```bash
lamplight config get
lamplight config set --library-path /mnt/media/books
lamplight config set --template "{author}/{title} ({year})"

# keep audiobooks separate from books (optional)
lamplight config set --audiobook-path /mnt/media/audiobooks

# if deluge runs in docker (see below)
lamplight config set --deluge-path /data --host-path /opt/docker/data/delugevpn/downloads
```

if `--audiobook-path` is set, mp3/m4b/m4a files get organized there instead of `--library-path`. if it's not set, everything goes to the same place like before.

available template tokens: `{author}`, `{title}`, `{year}`, `{publisher}`, `{isbn}`, `{format}`

### indexer
```bash
lamplight indexer list
lamplight indexer add --name myindexer --base-url http://... --api-key xxx
lamplight indexer delete myindexer
lamplight indexer update 1 --priority 10
```

### provider (Prowlarr / Jackett)
```bash
lamplight provider add --name prowlarr --type prowlarr --host 192.168.0.17 --port 9696 --api-key xxx
lamplight provider list
lamplight provider sync          # only syncs book-supporting indexers
lamplight provider sync --all    # syncs everything
lamplight provider delete prowlarr
```

### client (Deluge)
```bash
lamplight client add --name deluge --host 192.168.0.17 --port 8112 --password xxx
lamplight client list
lamplight client delete deluge
lamplight client update 1 --priority 1
```

---

## setup

```bash
# 1. add your provider
lamplight provider add --name prowlarr --type prowlarr --host 192.168.0.17 --port 9696 --api-key your_key

# 2. sync indexers (pulls only book-supporting ones)
lamplight provider sync

# 3. add deluge
lamplight client add --name deluge --type deluge --host 192.168.0.17 --port 8112 --password your_password

# 4. set your library path
lamplight config set --library-path /mnt/media/books

# 5. search and download
lamplight search "consider phlebas" -t epub
lamplight download 1

# 6. once it's done, sync status then organize
lamplight history sync
lamplight organize
```

---

## docker path mapping

If Deluge runs in a Docker container, it reports file paths from inside the container — not the real path on your host. This will cause `lamplight organize` to fail because the file doesn't exist at that path.

Fix it by telling lamplight how to translate:

```bash
lamplight config set \
  --deluge-path /data \
  --host-path /opt/docker/data/delugevpn/downloads
```

So if Deluge says a file is at `/data/incomplete/Some Book/Some Book.epub`, lamplight will look for it at `/opt/docker/data/delugevpn/downloads/incomplete/Some Book/Some Book.epub`.

To find your container's internal path: check your docker-compose volume mount for Deluge. Something like:

```yaml
volumes:
  - /opt/docker/data/delugevpn/downloads:/data
#    ^^^^ host path                        ^^^^ container path
```

The left side is your `--host-path`, the right side is your `--deluge-path`.

---

## building

```bash
go build -o target/lamplight
# or
./scripts/dev-build.sh
```

---

## testing

```bash
# run everything
go test ./...

# run a specific package
go test ./internal/util/...
go test ./internal/client/...
go test ./cmd/...

# with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

the test suite covers:
- **path template** — `ApplyTemplate`, `sanitizePathComponent`, `IsComplete` — all edge cases including special chars, truncation, unknown fields
- **metadata reading** — EPUB (Dublin Core OPF), MP3 ID3v2 (latin1, UTF-16 LE/BE, UTF-16BE without BOM), M4B iTunes atoms, filename fallback — all tested against programmatically built fixtures
- **organize logic** — single file, multi-file folder (stays together), single file unwrapping, conflict resolution (`_2`, `_3`), dry-run, uncategorized fallback, orphaned dir cleanup
- **history repository** — Save, FindAll ordering, ExistsByLink (exact match), FindActive (excludes empty hash), FindCompleted (requires file path), all update methods — all with in-memory SQLite
- **library config repository** — default creation, upsert, field updates — in-memory SQLite
- **deluge client** — Authenticate (success, bad password, server down), Add magnet, Add torrent file, GetTorrentStatus (single file, multi-file → folder path, no files fallback) — all against a local mock HTTP server
- **resolver** — direct magnet, redirect-to-magnet, torrent file download, HTML rejection, empty body, non-torrent content, 404
- **sync helpers** — `translatePath` (all prefix/no-match/empty cases), `delugeStateToStatus` (every Deluge state)
- **display utils** — `SmartTruncate`, `BytesToMb`, `CleanString`

---

## how format detection works

Results come with a `FORMAT` column. Detection runs in priority order:

1. `torznab:attr name="format"` — if the indexer sends it explicitly (Libgen does), done
2. Category ID — `3030` = audiobook, `7030` = comic
3. Title keywords — looks for things like `[EPUB]`, `.m4b`, `unabridged`, `[CBZ]` in the title

TPB and general indexers usually show `unknown` since they don't tag format. Book-specific indexers like Libgen and MyAnonaMouse give you real format data.
