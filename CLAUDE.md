# Lamplight CLI

A Go CLI tool for searching torrents across multiple indexer providers (Prowlarr/Jackett) and downloading them to configured downloader clients (Deluge).

## Architecture

```
main.go                          # Entry point → cmd.Execute()
cmd/                             # Cobra CLI commands
internal/
  client/                        # HTTP clients (Torznab, Deluge, Jackett, Prowlarr)
  constants/                     # Filter category constants (Books)
  dao/                           # Data transfer objects (SearchRequest, SearchResult, etc.)
  domain/
    entity/                      # GORM entities (Indexer, Provider, Downloader, SearchCache)
    repository/                  # Data access layer (one repo per entity)
  service/                       # Business logic (SearchService, SearchBackend, filtering)
  util/                          # DB init, display formatting, XML parsing, query building
```

## Key Patterns

- **Repository pattern** for all DB access via GORM + SQLite
- **Strategy pattern** for SearchBackend (Torznab), ProviderClient (Jackett/Prowlarr), DownloaderClient (Deluge)
- **Cobra CLI** with nested subcommands: `lamplight {indexer,client,provider} {add,list,delete,update,sync}`
- **Search cache** is a singleton row (ID=1) storing JSON blob of last search results
- Database location is platform-specific (macOS: `~/Library/Application Support/lamplight-cli/`)
- DB is opened and auto-migrated on every command invocation

## Commands

| Command | Status |
|---------|--------|
| `search <query>` | Working (flags: --indexer, --limit, --books) |
| `download <index>` | Working but not fully vetted |
| `indexer add/list/delete` | Working (no update) |
| `client add/list/delete/update` | Complete CRUD |
| `provider add/list/sync` | Working (no delete, no update cmd wired) |

## Building

```bash
./scripts/dev-build.sh    # outputs to target/lamplight
go build -o target/lamplight
```

## Database

SQLite with 4 tables: `indexer`, `downloader`, `provider`, `search_cache`
- Auto-migrated on every `utils.Open()` call
- DB path: `lamplight-cli.db.sqlite` in platform data dir

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `gorm.io/gorm` + `gorm.io/driver/sqlite` - ORM
- Standard library for HTTP, XML parsing, JSON

## Testing

No tests exist yet. When adding tests:
- Use `cmd.OutOrStdout()` pattern already in place for testable output
- Repositories accept `*gorm.DB` — use in-memory SQLite for tests
- HTTP clients accept `*http.Client` — use `httptest.Server` for mocking

## Style Notes

- Error wrapping with `fmt.Errorf("context: %w", err)` throughout
- Flag shorthand letters vary (e.g., `-o` for host, `-w` for password)
- `normalizeName()` in addClient.go sanitizes user input for names
- `--unsafe` flag on list commands reveals API keys/passwords
