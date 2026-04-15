# contributing

fork the repo, make your changes on a branch, open a PR against `main`. CI runs automatically — lint and tests need to pass before anything gets merged.

```bash
git clone https://github.com/<your-username>/lamplight-cli.git
cd lamplight-cli
git checkout -b my-feature
# make changes
go test ./...
git push origin my-feature
# open PR on GitHub
```

keep PRs focused — one thing at a time is easier to review than a big pile of changes.
