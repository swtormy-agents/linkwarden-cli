---
name: linkwarden
description: Read and manage Linkwarden bookmarks through the linkwarden CLI. Use for Linkwarden collections, tags, link search, or REST API automation from the shell.
---

# linkwarden CLI

Unofficial client for the [Linkwarden REST API](https://docs.linkwarden.app/api). Output is JSON on stdout; errors are JSON on stderr.

## Configuration

Set credentials in the environment or in a config file (`chmod 600`):

```bash
export LINKWARDEN_URL=https://linkwarden.example
export LINKWARDEN_TOKEN=your_jwt_here
```

Default config path: `$XDG_CONFIG_HOME/linkwarden/config.json` (Windows: `%AppData%\\linkwarden\\config.json`). Override with `LINKWARDEN_CONFIG` or `LINKWARDEN_CONFIG_DIR`.

The token is sent only to `LINKWARDEN_URL` as `Authorization: Bearer`. It is never printed; `--dry-run` and errors use `<TOKEN>`.

## Read commands

```bash
linkwarden links list --cursor 0
linkwarden links search "kubernetes"
linkwarden links get 12
linkwarden collections list
linkwarden tags list
linkwarden version
```

Use `--dry-run` to preview a request without sending it.

## Writes (opt-in)

Create, update, and delete are blocked unless the human enables writes (`LINKWARDEN_ALLOW_WRITE=true`, `"allow_write": true` in config, or `--allow-write` on one command). Confirm with the user before running write commands.

```bash
linkwarden links create --allow-write --json '{"url":"https://example.com","name":"Example"}'
linkwarden links update 12 --allow-write --json '{"name":"Updated title"}'
linkwarden links delete 12 --allow-write
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API or runtime failure |
| 2 | Usage error |
| 3 | Auth failure |
| 4 | Blocked by allow_write policy |

Install the skill file: `linkwarden skill > skills/linkwarden/SKILL.md` or copy from this repository.
