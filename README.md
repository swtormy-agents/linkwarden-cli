# linkwarden-cli

Unofficial CLI for [Linkwarden](https://github.com/linkwarden/linkwarden) — a self-hosted collaborative bookmark manager. This project is **independent, unofficial, and not affiliated with, endorsed by, or maintained by** Linkwarden or its authors. It talks to your instance over the [Linkwarden REST API](https://docs.linkwarden.app/api).

Motivation: [linkwarden#1572](https://github.com/linkwarden/linkwarden/issues/1572) — expose bookmarks to agents without running a separate MCP server.

## Install

### Download a binary from GitHub Releases

1. Open [GitHub Releases](https://github.com/swtormy-agents/linkwarden-cli/releases) and download the archive for your OS/arch (`linkwarden_<version>_linux_amd64.tar.gz`, `darwin_arm64`, `windows_amd64.zip`, etc.).
2. Verify the SHA256 checksum against `checksums.txt` on the same release:

```bash
sha256sum -c checksums.txt --ignore-missing   # Linux
# or
certutil -hashfile linkwarden.exe SHA256        # Windows
```

3. Extract the archive. It contains `linkwarden` (or `linkwarden.exe`), `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES`, and full dependency license texts under `third_party_licenses/`.

Releases are published as **drafts** first; pick the latest draft or published release from the maintainer.

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/swtormy-agents/linkwarden-cli/main/scripts/install.sh | bash
```

Environment variables:

- `LINKWARDEN_VERSION` — release tag (default: `latest`, resolves via GitHub API)
- `LINKWARDEN_INSTALL_DIR` — install directory (default: `/usr/local/bin` if writable, else `~/.local/bin`)

The script downloads the correct archive (`.zip` on Windows), verifies `checksums.txt`, and installs the binary.

### Go install

```bash
go install github.com/swtormy-agents/linkwarden-cli/cmd/linkwarden@latest
```

### Build from source

```bash
git clone https://github.com/swtormy-agents/linkwarden-cli.git
cd linkwarden-cli
go build -o linkwarden ./cmd/linkwarden
```

## Authentication

Linkwarden uses `Authorization: Bearer <JWT>` (see [API docs](https://docs.linkwarden.app/api/api-introduction)). Create a session token via `POST /api/v1/session` or use a token from your deployment’s settings flow.

1. Set your instance URL and token — **never pass the token on the command line** (shell history and process listings):

```bash
export LINKWARDEN_URL=https://linkwarden.example
export LINKWARDEN_TOKEN=your_jwt_here
```

Alternatively, store credentials in a config file (file mode `600`, directory mode `700`):

```json
{
  "url": "https://linkwarden.example",
  "token": "your_jwt_here"
}
```

Default path: `$XDG_CONFIG_HOME/linkwarden/config.json` (on Windows, `%AppData%\\linkwarden\\config.json` via `os.UserConfigDir()`). Override with `LINKWARDEN_CONFIG` (single file) or `LINKWARDEN_CONFIG_DIR` (directory containing `config.json` and optional `allow_write`).

The CLI **never prints** your token. Errors and `--dry-run` output mask it as `<TOKEN>`. Requests are sent **only** to the configured `LINKWARDEN_URL` (must be `http://` or `https://` with a host; userinfo in the URL is rejected; `http://` logs a warning).

## Usage

JSON on stdout; failures are one JSON object on stderr. Filter with `--jq` or `--pretty`.

```bash
linkwarden links list --cursor 0
linkwarden links search "docker compose"
linkwarden links get 42
linkwarden collections list
linkwarden tags list
linkwarden version
linkwarden skill
```

Preview a request without sending it:

```bash
linkwarden links create --dry-run --json '{"url":"https://example.com"}'
```

## Write access (`allow_write`)

`links create`, `links update`, and `links delete` are **read-only by default**. Enable writes in one of these ways (highest precedence first):

1. **`--allow-write`** on a single command.
2. **`LINKWARDEN_ALLOW_WRITE=true`** (or `1`, `yes`, `on`, `write`).
3. File **`allow_write`** containing `true` in the config directory.
4. **`"allow_write": true`** in `config.json`.

Each enabled write prints a one-time stderr notice describing which source was used.

## Agent skill

Bundled skill for coding agents:

```bash
linkwarden skill
```

Source in this repository: `skills/linkwarden/SKILL.md`.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API or runtime failure |
| 2 | Usage error |
| 3 | Auth failure |
| 4 | Blocked by allow_write policy |

## License

Apache License 2.0 — see [LICENSE](LICENSE). Third-party licenses: [NOTICE](NOTICE), [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).
