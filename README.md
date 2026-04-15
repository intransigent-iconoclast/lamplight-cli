# lamplight-cli

A CLI for finding and downloading books. Talks to your self-hosted Prowlarr or Jackett instance, searches across all your configured indexers, and pipes results straight to Deluge. That's it.

Built in Go. Open source. MIT licensed.

---

## Requirements

lamplight is a frontend — it needs these self-hosted services running before it can do anything:

### Required

**[Deluge](https://deluge-torrent.org/)** — the download client. lamplight sends torrents here and polls it for status.
- Needs the web UI enabled (Settings → Interface → Enable Web Interface)
- Default port: `8112`
- Works fine in Docker — see [Docker path mapping](#docker-path-mapping) if your paths differ

**[Prowlarr](https://github.com/Prowlarr/Prowlarr) or [Jackett](https://github.com/Jackett/Jackett)** — the indexer manager. lamplight queries this to search across all your configured torrent indexers.
- Prowlarr is recommended (actively maintained, better API)
- Jackett works too
- You need at least one book-supporting indexer configured (e.g. Libgen, MyAnonaMouse, IPTorrents)
- Default Prowlarr port: `9696` — default Jackett port: `9117`

### Optional but recommended

**[Calibre](https://calibre-ebook.com/)** — not required to run lamplight, but if you manage your library with Calibre you can set the template to match Calibre's folder layout so they play nicely together:

```bash
lamplight config set --template "{author}/{title}/{title} - {author}"
```

---

## Install

### From a release (recommended)

Download the latest binary for your platform from the [releases page](https://github.com/intransigent-iconoclast/lamplight-cli/releases).

| Platform | Archive |
|----------|---------|
| Linux (amd64) | `lamplight_vX.X.X_linux_amd64.tar.gz` |
| macOS (Apple Silicon) | `lamplight_vX.X.X_darwin_arm64.tar.gz` |
| macOS (Intel) | `lamplight_vX.X.X_darwin_amd64.tar.gz` |
| Windows (amd64) | `lamplight_vX.X.X_windows_amd64.zip` |

```bash
# Example: Linux
tar xzf lamplight_v*.tar.gz
sudo mv lamplight /usr/local/bin/

# Example: macOS
tar xzf lamplight_v*.tar.gz
mv lamplight /usr/local/bin/
```

### From source

Requires Go 1.24+ and a C compiler (lamplight uses CGO for SQLite — see [CONTRIBUTING.md](CONTRIBUTING.md) for platform-specific setup).

```bash
git clone https://github.com/intransigent-iconoclast/lamplight-cli.git
cd lamplight-cli
go build -o lamplight .
```

---

## What it does

- Search for books across multiple torrent indexers at once
- Filter results by format — epub, pdf, mobi, audiobook, comic, or just `book` to catch all prose formats
- Sort by seeders, leechers, size, or title
- Download straight to Deluge with one command
- Tracks download history and syncs status from Deluge
- Organizes completed downloads into your library automatically — reads embedded metadata from epub, mp3, and m4b files
- Handles multi-file downloads (audiobook chapters, comic bundles) as a group
- Syncs indexers from Prowlarr/Jackett automatically, filtering to only book-supporting ones

---

## Commands

### search
```bash
lamplight search "consider phlebas"
lamplight search "dune" -l 40 -t epub
lamplight search "one piece" -t comic -s seeders
lamplight search "stephen king" -t book,audiobook -l 100
```

| Flag | What it does |
|------|-------------|
| `-l` | How many results to fetch (default 15, use 0 for no limit) |
| `-t` | Filter by type: `book`, `epub`, `pdf`, `mobi`, `audiobook`, `comic`, `unknown`, `all` — comma-separated works |
| `-s` | Sort by: `seeders` (default), `leechers`, `size`, `title` |
| `-b` | Book category filter, on by default |
| `-i` | Search a specific indexer by index |

Results are cached for 30 minutes. After that you'll need to re-run your search.

### download
```bash
lamplight download 3            # downloads result #3 from your last search
lamplight download 3 --force    # re-download even if it's already in history, or search results are stale
```

lamplight blocks duplicate downloads — if that link is already in your history it'll tell you. Use `--force` to download anyway.

### history
```bash
lamplight history list
lamplight history list --filter failed
lamplight history list "dune"                     # search by title (substring)
lamplight history list "herbert" --filter completed

lamplight history sync                            # poll Deluge for status updates
lamplight history sync -w                         # live progress view, refreshes every second
lamplight history retry 3                         # re-send entry #3 to Deluge
lamplight history retry --all-failed              # re-send everything that failed at once
lamplight history cancel 3                        # remove from Deluge and history
lamplight history cancel 3 --delete-data          # same, but also deletes files from disk
lamplight history update 3 --status failed        # manually fix a stuck entry
lamplight history clear
```

The index shown in `history list` is always the global index — use it directly with `retry`, `update`, or `cancel` even when filtering.

### organize
```bash
lamplight organize                                # process all completed downloads
lamplight organize ~/Downloads/some-book.epub     # one-off manual file or folder
lamplight organize --dry-run                      # preview without moving anything
```

Run `history sync` first to make sure statuses are up to date, then `organize` to move everything into your library.

Files with complete metadata (author + title) go into:
```
<library-path>/<template>.<ext>
```

Everything else ends up in:
```
<library-path>/uncategorized/<filename>
```

**Multi-file downloads** (audiobook chapters, comic bundles) are kept together in a folder:
```
<library-path>/<author>/<title>/01 - Chapter One.mp3
                                02 - Chapter Two.mp3
```

If you have both an ebook and an audiobook of the same title, they sit side by side under the same author folder without conflicting:
```
/mnt/media/books/
  Frank Herbert/
    Dune/
      Dune - Frank Herbert.epub
      Dune - Frank Herbert.mobi
```

If you set `--audiobook-path`, audiobooks go to a completely separate root:
```
/mnt/media/books/Frank Herbert/Dune/Dune - Frank Herbert.epub
/mnt/media/audiobooks/Frank Herbert/Dune/01.mp3
```

If a filename already exists, lamplight appends `_2`, `_3`, etc. rather than overwriting.

**Metadata sources**, in order of preference:
- epub: Dublin Core fields from the OPF file inside the zip
- mp3: ID3v2 tags (supports UTF-8, UTF-16 LE/BE)
- m4b/m4a: iTunes MP4 atoms (`©nam`, `©ART`, `©day`)
- Fallback: `Author - Title` filename pattern

### config
```bash
lamplight config get
lamplight config set --library-path /mnt/media/books
lamplight config set --template "{author}/{title} ({year})"

# Keep audiobooks separate from books (optional)
lamplight config set --audiobook-path /mnt/media/audiobooks

# If Deluge runs in Docker (see below)
lamplight config set --deluge-path /data --host-path /opt/docker/data/delugevpn/downloads
```

If `--audiobook-path` is set, mp3/m4b/m4a files get organized there instead of `--library-path`. If it's not set, everything goes to the same place.

Available template tokens: `{author}`, `{title}`, `{year}`, `{publisher}`, `{isbn}`, `{format}`

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
lamplight client add --name deluge --client-type deluge --host 192.168.0.17 --port 8112 --password xxx
lamplight client list
lamplight client delete deluge
lamplight client update 1 --priority 1
```

---

## Setup

```bash
# 1. Add your provider
lamplight provider add --name prowlarr --type prowlarr --host 192.168.0.17 --port 9696 --api-key your_key

# 2. Sync indexers (pulls only book-supporting ones)
lamplight provider sync

# 3. Add Deluge
lamplight client add --name deluge --client-type deluge --host 192.168.0.17 --port 8112 --password your_password

# 4. Set your library path
lamplight config set --library-path /mnt/media/books

# 5. Search and download
lamplight search "consider phlebas" -t epub
lamplight download 1

# 6. Once it's done, sync status then organize
lamplight history sync
lamplight organize
```

---

## Docker path mapping

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

## Building

```bash
go build -o target/lamplight
# or
./scripts/dev-build.sh
```

---

## Testing

```bash
# Run everything
go test ./...

# Run a specific package
go test ./internal/util/...
go test ./internal/client/...
go test ./cmd/...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

The test suite covers:
- **Path template** — `ApplyTemplate`, `sanitizePathComponent`, `IsComplete` — all edge cases including special chars, truncation, unknown fields
- **Metadata reading** — EPUB (Dublin Core OPF), MP3 ID3v2 (latin1, UTF-16 LE/BE, UTF-16BE without BOM), M4B iTunes atoms, filename fallback — all tested against programmatically built fixtures
- **Organize logic** — single file, multi-file folder (stays together), single file unwrapping, conflict resolution (`_2`, `_3`), dry-run, uncategorized fallback, orphaned dir cleanup
- **History repository** — Save, FindAll ordering, ExistsByLink (exact match), FindActive (excludes empty hash), FindCompleted (requires file path), all update methods — all with in-memory SQLite
- **Library config repository** — default creation, upsert, field updates — in-memory SQLite
- **Deluge client** — Authenticate (success, bad password, server down), Add magnet, Add torrent file, GetTorrentStatus (single file, multi-file → folder path, no files fallback) — all against a local mock HTTP server
- **Resolver** — direct magnet, redirect-to-magnet, torrent file download, HTML rejection, empty body, non-torrent content, 404
- **Sync helpers** — `translatePath` (all prefix/no-match/empty cases), `delugeStateToStatus` (every Deluge state)
- **Display utils** — `SmartTruncate`, `BytesToMb`, `CleanString`

---

## How format detection works

Results come with a `FORMAT` column. Detection runs in priority order:

1. `torznab:attr name="format"` — if the indexer sends it explicitly (Libgen does), done
2. Category ID — `3030` = audiobook, `7030` = comic
3. Title keywords — looks for things like `[EPUB]`, `.m4b`, `unabridged`, `[CBZ]` in the title

TPB and general indexers usually show `unknown` since they don't tag format. Book-specific indexers like Libgen and MyAnonaMouse give you real format data.

---

## Contributing

Fork → branch → PR against `main`. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
