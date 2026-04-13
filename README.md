# lamplight-cli

A CLI for finding and downloading books. Talks to your self-hosted Prowlarr or Jackett instance, searches across all your configured indexers, and pipes results straight to Deluge. That's it.

Built in Go. Open source.

---

## what it does

- Search for books across multiple torrent indexers at once
- Filter results by format — epub, pdf, mobi, audiobook, comic, or just `book` to catch all prose formats
- Sort by seeders, leechers, size, or title
- Download straight to Deluge with one command
- Tracks download history
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

### download
```bash
lamplight download 3   # downloads result #3 from your last search
```

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

### history
```bash
lamplight history list
lamplight history clear
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

# 4. search and download
lamplight search "consider phlebas" -t epub
lamplight download 1
```

---

## building

```bash
go build -o target/lamplight
# or
./scripts/dev-build.sh
```

---

## how format detection works

Results come with a `FORMAT` column. Detection runs in priority order:

1. `torznab:attr name="format"` — if the indexer sends it explicitly (Libgen does), done
2. Category ID — `3030` = audiobook, `7030` = comic
3. Title keywords — looks for things like `[EPUB]`, `.m4b`, `unabridged`, `[CBZ]` in the title

TPB and general indexers usually show `unknown` since they don't tag format. Book-specific indexers like Libgen and MyAnonaMousegive you real format data.
