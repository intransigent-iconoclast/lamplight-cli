# Contributing

PRs are welcome. lamplight uses the forking workflow — if you're not familiar with it, [this guide](https://www.atlassian.com/git/tutorials/comparing-workflows/forking-workflow) covers it well.

## Project-specific requirements

lamplight uses CGO for SQLite, so you'll need a C compiler on your machine before anything will build:

- **macOS** — `xcode-select --install`
- **Linux** — `sudo apt install build-essential` (or equivalent for your distro)
- **Windows** — install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

Go 1.24+ is also required.

## Before opening a PR

```bash
go build ./...       # make sure it builds
go test ./...        # all tests pass
golangci-lint run    # no lint issues
```

CI checks all three automatically — a PR won't get merged if any of them are red.
