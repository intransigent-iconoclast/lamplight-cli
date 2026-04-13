# Security Policy

## Reporting a Vulnerability

If you find a security vulnerability in lamplight, please report it responsibly.

**Do not open a public issue.**

Instead, email **security@intransigent-iconoclast.dev** or use [GitHub's private vulnerability reporting](https://github.com/intransigent-iconoclast/lamplight-cli/security/advisories/new).

Please include:
- A description of the vulnerability
- Steps to reproduce
- Potential impact

You should expect an initial response within 72 hours.

## Scope

This project communicates with user-configured services (Prowlarr, Jackett, Deluge) over HTTP. It stores configuration and history in a local SQLite database. Security concerns include but are not limited to:

- API key or credential leakage
- Command injection via user input
- Path traversal in file organization
- Unsafe handling of torrent/magnet data
