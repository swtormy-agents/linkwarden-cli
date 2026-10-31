#!/usr/bin/env bash
# Install linkwarden from GitHub Releases (draft or published).
#   curl -fsSL https://raw.githubusercontent.com/swtormy-agents/linkwarden-cli/v0.1.0/scripts/install.sh | bash
# Environment:
#   LINKWARDEN_VERSION     release tag to install (default: latest)
#   LINKWARDEN_INSTALL_DIR target directory (default: /usr/local/bin if writable, else ~/.local/bin)
set -euo pipefail

repo="swtormy-agents/linkwarden-cli"
version="${LINKWARDEN_VERSION:-${1:-latest}}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  MINGW* | MSYS* | CYGWIN* | Windows*) os=windows ;;
  *)
    echo "Unsupported OS: $(uname -s). Download manually: https://github.com/${repo}/releases" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [[ -n "${LINKWARDEN_INSTALL_DIR:-}" ]]; then
  dir="$LINKWARDEN_INSTALL_DIR"
elif [[ -w /usr/local/bin ]]; then
  dir=/usr/local/bin
else
  dir="$HOME/.local/bin"
fi

resolve_tag() {
  local v="$1"
  if [[ "$v" == "latest" ]]; then
    curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
  else
    if [[ "$v" != v* ]]; then
      echo "v${v}"
    else
      echo "$v"
    fi
  fi
}

tag="$(resolve_tag "$version")"
if [[ -z "$tag" ]]; then
  echo "Could not resolve release tag (version=${version})" >&2
  exit 1
fi

ver="${tag#v}"
base="https://github.com/${repo}/releases/download/${tag}"

if [[ "$os" == "windows" ]]; then
  asset="linkwarden_${ver}_${os}_${arch}.zip"
  bin_name="linkwarden.exe"
else
  asset="linkwarden_${ver}_${os}_${arch}.tar.gz"
  bin_name="linkwarden"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset} (${tag})"
curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt"

want="$(grep -F " ${asset}" "${tmp}/checksums.txt" | awk '{print $1}' | head -n1)"
if [[ -z "$want" ]]; then
  echo "No checksum for ${asset} in checksums.txt" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "${tmp}/${asset}" | awk '{print $1}')"
else
  got="$(shasum -a 256 "${tmp}/${asset}" | awk '{print $1}')"
fi
if [[ "$want" != "$got" ]]; then
  echo "Checksum mismatch for ${asset}" >&2
  exit 1
fi

if [[ "$asset" == *.zip ]]; then
  unzip -q "${tmp}/${asset}" -d "${tmp}/extract"
else
  mkdir -p "${tmp}/extract"
  tar -xzf "${tmp}/${asset}" -C "${tmp}/extract"
fi

bin=""
if [[ -f "${tmp}/extract/${bin_name}" ]]; then
  bin="${tmp}/extract/${bin_name}"
elif [[ -f "${tmp}/extract/linkwarden_${ver}_${os}_${arch}/${bin_name}" ]]; then
  bin="${tmp}/extract/linkwarden_${ver}_${os}_${arch}/${bin_name}"
else
  bin="$(find "${tmp}/extract" -name "${bin_name}" -type f | head -n1)"
fi
if [[ -z "$bin" || ! -f "$bin" ]]; then
  echo "Could not find ${bin_name} in archive" >&2
  exit 1
fi

mkdir -p "$dir"
install -m 0755 "$bin" "${dir}/${bin_name}"
echo "Installed ${dir}/${bin_name}"
case ":$PATH:" in *":${dir}:"*) ;; *) echo "Add ${dir} to your PATH." ;; esac
"${dir}/${bin_name}" version
