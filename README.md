# lamplight-cli

A CLI for finding and downloading books. Talks to your self-hosted Prowlarr or Jackett instance, searches across all your configured indexers, and pipes results straight to Deluge. That's it.

Built in Go. Open source. MIT licensed.

- Search across multiple torrent indexers at once
- Filter by format — epub, pdf, mobi, audiobook, comic
- Download straight to Deluge with one command
- Tracks history and syncs status from Deluge
- Organizes completed downloads into your library — reads embedded metadata from epub, mp3, and m4b files
- Handles multi-file downloads (audiobook chapters, comic bundles) as a group

---

## Requirements

lamplight is a frontend — it needs these running before it can do anything:

**[Deluge](https://deluge-torrent.org/)** — the download client.
- Needs the web UI enabled (Settings → Interface → Enable Web Interface)
- Default port: `8112`
- Running in Docker? See [Docker path mapping](#docker-path-mapping)

**[Prowlarr](https://github.com/Prowlarr/Prowlarr) or [Jackett](https://github.com/Jackett/Jackett)** — the indexer manager.
- Prowlarr is recommended (actively maintained, better API)
- You need at least one book-supporting indexer configured (e.g. Libgen, MyAnonaMouse, IPTorrents)
- Default Prowlarr port: `9696` — default Jackett port: `9117`

**[Calibre](https://calibre-ebook.com/)** *(optional)* — if you manage your library with Calibre, you can match its folder layout exactly:
```bash
lamplight config set --template "{author}/{title}/{title} - {author}"
```

---

## Install

### From a release (recommended)

Download the latest binary from the [releases page](https://github.com/intransigent-iconoclast/lamplight-cli/releases).

| Platform | Archive |
|----------|---------|
| Linux (amd64) | `lamplight_vX.X.X_linux_amd64.tar.gz` |
| macOS | `lamplight_vX.X.X_darwin_arm64.tar.gz` |
| Windows (amd64) | `lamplight_vX.X.X_windows_amd64.zip` |

> **Intel Mac?** Rosetta 2 runs the ARM binary transparently — no extra steps needed.

```bash
# Linux
tar xzf lamplight_v*.tar.gz
sudo mv lamplight /usr/local/bin/

# macOS
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

## Quick start

First time setup — do this once:

```bash
# 1. Add your provider (Prowlarr or Jackett)
lamplight provider add --name prowlarr --type prowlarr --host 192.168.0.17 --port 9696 --api-key your_key

# 2. Pull book-supporting indexers from it
lamplight provider sync

# 3. Add Deluge
lamplight client add --name deluge --client-type deluge --host 192.168.0.17 --port 8112 --password your_password

# 4. Set where your library lives
lamplight config set --library-path /mnt/media/books

# Optional: keep audiobooks in a separate folder
lamplight config set --audiobook-path /mnt/media/audiobooks
```

---

## Standard workflow

This is the loop you'll run every time you want something:

```bash
# 1. Search
lamplight search "consider phlebas" -t epub

# 2. Download — use the index from the search results
lamplight download 3

# 3. Check status (add -w for a live progress bar)
lamplight history sync -w

# 4. Once it's done, move it into your library
lamplight organize
```

---

## Commands

### search

```bash
lamplight search "dune"
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
lamplight download 3 --force    # re-download even if it's already in history
```

lamplight blocks duplicate downloads — if that link is already in your history it'll tell you. Use `--force` to override.

### history

```bash
lamplight history list
lamplight history list --filter failed
lamplight history list "dune"                     # search by title (substring)
lamplight history list "herbert" --filter completed

lamplight history sync                            # poll Deluge for status updates
lamplight history sync -w                         # live progress bar, refreshes every second

lamplight history retry 3                         # re-send entry #3 to Deluge
lamplight history retry --all-failed              # re-send everything that failed

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

Files with complete metadata (author + title) go into:
```
<library-path>/<template>.<ext>
```

Everything else ends up in:
```
<library-path>/uncategorized/<filename>
```

**Multi-file downloads** (audiobook chapters, comic bundles) are kept together as a folder:
```
<library-path>/<author>/<title>/01 - Chapter One.mp3
                                02 - Chapter Two.mp3
```

If a filename already exists, lamplight appends `_2`, `_3`, etc. rather than overwriting.

**Metadata sources**, in order of preference:
- epub: Dublin Core fields from the OPF file inside the zip
- mp3: ID3v2 tags (supports UTF-8, UTF-16 LE/BE)
- m4b/m4a: iTunes MP4 atoms (`©nam`, `©ART`, `©day`)
- Fallback: `Author - Title` filename pattern

---

## Configuration

```bash
lamplight config get
lamplight config set --library-path /mnt/media/books
lamplight config set --template "{author}/{title} ({year})"
lamplight config set --audiobook-path /mnt/media/audiobooks

# Docker path translation (see Docker path mapping below)
lamplight config set --deluge-path /data --host-path /opt/docker/data/delugevpn/downloads
```

Available template tokens: `{author}`, `{title}`, `{year}`, `{publisher}`, `{isbn}`, `{format}`

If `--audiobook-path` is set, mp3/m4b/m4a files go there instead of `--library-path`.

---

## Management

These are one-time setup commands you'll rarely need after the initial configuration.

### Providers (Prowlarr / Jackett)

```bash
lamplight provider list
lamplight provider add --name prowlarr --type prowlarr --host 192.168.0.17 --port 9696 --api-key xxx
lamplight provider sync          # only syncs book-supporting indexers
lamplight provider sync --all    # syncs everything
lamplight provider delete 1
```

### Indexers

```bash
lamplight indexer list
lamplight indexer add --name myindexer --base-url http://... --api-key xxx
lamplight indexer delete --index 1
lamplight indexer update 1 --priority 10
```

### Download clients (Deluge)

```bash
lamplight client list
lamplight client add --name deluge --client-type deluge --host 192.168.0.17 --port 8112 --password xxx
lamplight client delete 1
lamplight client update 1 --priority 1
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

To find the paths, check your docker-compose volume mount for Deluge:

```yaml
volumes:
  - /opt/docker/data/delugevpn/downloads:/data
#    ^^^^ host path (--host-path)          ^^^^ container path (--deluge-path)
```

---

## How format detection works

Results come with a `FORMAT` column. Detection runs in priority order:

1. `torznab:attr name="format"` — if the indexer sends it explicitly (Libgen does), done
2. Category ID — `3030` = audiobook, `7030` = comic
3. Title keywords — looks for things like `[EPUB]`, `.m4b`, `unabridged`, `[CBZ]` in the title

TPB and general indexers usually show `unknown` since they don't tag format. Book-specific indexers like Libgen and MyAnonaMouse give you real format data.

---

## Building

```bash
go build -o target/lamplight
# or
./scripts/dev-build.sh
```

## Testing

```bash
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## Contributing

Fork → branch → PR against `main`. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
