# Knox - Vault Fleeting Note Expiry Manager

A Go CLI tool that automatically manages fleeting notes in your Obsidian vault. Marks notes for deletion based on expiry dates in frontmatter, then cleans them up automatically.

## Usage

```bash
go build
./knox -vault /path/to/vault [-dry-run] [-db knox.db]
```

### Flags
- `-vault`: Path to your Obsidian vault (defaults to `$HOME/vault`)
- `-dry-run`: Show what would be deleted without actually deleting
- `-db`: Path to SQLite database (defaults to `knox.db`)

## Frontmatter Format

Add an `expiry_date` field to fleeting notes in your inbox:

```yaml
---
title: Quick thought
expiry_date: 7d
---

Some temporary note content...
```

Supported formats:
- `7d` - 7 days
- `30d` - 30 days
- `2h` - 2 hours
- `5m` - 5 minutes

## How it works

1. Scans the `/inbox` folder in your vault
2. Finds notes with `expiry_date` in frontmatter
3. Stores tracking info in local SQLite database
4. On each run, checks for expired notes and deletes them
5. Removes deleted notes from database

Notes without an `expiry_date` field are ignored completely.
