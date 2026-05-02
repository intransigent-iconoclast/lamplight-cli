#!/usr/bin/env sh
# lamplight installer
# Usage: curl -fsSL https://raw.githubusercontent.com/intransigent-iconoclast/lamplight-cli/main/install.sh | sh
#
# Options (set as env vars before running):
#   LAMPLIGHT_VERSION     install a specific version (default: latest)
#   LAMPLIGHT_INSTALL_DIR install to a specific directory (default: ~/.local/bin)
#
set -eu

REPO="intransigent-iconoclast/lamplight-cli"
BINARY="lamplight"

# --- colors (only when stdout is a terminal) ---
if [ -t 2 ]; then
    BOLD="\033[1m"
    GREEN="\033[32m"
    YELLOW="\033[33m"
    RED="\033[31m"
    RESET="\033[0m"
else
    BOLD="" GREEN="" YELLOW="" RED="" RESET=""
fi

info()    { printf "${GREEN}==>${RESET} ${BOLD}%s${RESET}\n" "$*" >&2; }
warn()    { printf "${YELLOW}warning:${RESET} %s\n" "$*" >&2; }
error()   { printf "${RED}error:${RESET} %s\n" "$*" >&2; exit 1; }
success() { printf "${GREEN}✓${RESET} ${BOLD}%s${RESET}\n" "$*" >&2; }

# --- clean up temp dir on exit ---
TMPDIR_WORK=""
cleanup() { [ -n "$TMPDIR_WORK" ] && rm -rf "$TMPDIR_WORK"; }
trap cleanup EXIT INT TERM

# --- os detection ---
get_os() {
    case "$(uname -s)" in
        Linux)  echo "linux" ;;
        Darwin) echo "darwin" ;;
        *)      error "unsupported OS: $(uname -s). Windows users: download from https://github.com/${REPO}/releases" ;;
    esac
}

# --- arch detection ---
get_arch() {
    case "$(uname -m)" in
        x86_64)        echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "unsupported architecture: $(uname -m)" ;;
    esac
}

# --- download with curl or wget ---
download() {
    url="$1"; dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        error "curl or wget is required"
    fi
}

fetch_text() {
    url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url"
    else
        error "curl or wget is required"
    fi
}

# --- resolve latest release tag ---
get_latest_version() {
    fetch_text "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/'
}

# --- sha256 checksum verification ---
verify_checksum() {
    file="$1"; checksum_file="$2"
    filename="$(basename "$file")"
    expected="$(grep "$filename" "$checksum_file" | awk '{print $1}')"

    if [ -z "$expected" ]; then
        warn "no checksum found for $filename — skipping verification"
        return
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "$file" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    else
        warn "sha256sum/shasum not available — skipping checksum verification"
        return
    fi

    if [ "$expected" != "$actual" ]; then
        error "checksum mismatch for $filename\n  expected: $expected\n  actual:   $actual"
    fi

    info "Checksum verified"
}

# --- pick install directory ---
pick_install_dir() {
    if [ -n "${LAMPLIGHT_INSTALL_DIR:-}" ]; then
        echo "$LAMPLIGHT_INSTALL_DIR"
        return
    fi

    local_bin="$HOME/.local/bin"
    mkdir -p "$local_bin" 2>/dev/null && echo "$local_bin" && return

    echo "/usr/local/bin"
}

# --- main ---
main() {
    os="$(get_os)"
    arch="$(get_arch)"

    # Intel Macs: use the arm64 binary — Rosetta 2 runs it transparently
    if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
        arch="arm64"
    fi

    info "Resolving latest version..."
    version="${LAMPLIGHT_VERSION:-$(get_latest_version)}"
    [ -n "$version" ] || error "could not resolve latest version — check your connection or set LAMPLIGHT_VERSION"

    archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
    base_url="https://github.com/${REPO}/releases/download/${version}"

    info "Installing ${BINARY} ${version} (${os}/${arch})"

    TMPDIR_WORK="$(mktemp -d)"

    info "Downloading..."
    download "${base_url}/${archive}" "${TMPDIR_WORK}/${archive}"
    download "${base_url}/checksums.txt" "${TMPDIR_WORK}/checksums.txt"

    verify_checksum "${TMPDIR_WORK}/${archive}" "${TMPDIR_WORK}/checksums.txt"

    info "Extracting..."
    tar -xzf "${TMPDIR_WORK}/${archive}" -C "$TMPDIR_WORK"

    install_dir="$(pick_install_dir)"
    install_path="${install_dir}/${BINARY}"

    # if it already exists, show what's being replaced
    if command -v "$BINARY" >/dev/null 2>&1; then
        current="$(${BINARY} version 2>/dev/null || echo 'unknown')"
        info "Replacing existing install (${current})"
    fi

    mkdir -p "$install_dir"
    mv "${TMPDIR_WORK}/${BINARY}" "$install_path"
    chmod +x "$install_path"

    success "Installed ${BINARY} ${version} → ${install_path}"

    # warn if install dir isn't on PATH
    case ":${PATH}:" in
        *":${install_dir}:"*) ;;
        *)
            printf "\n"
            warn "${install_dir} is not in your PATH. Add this to your shell config:"
            printf "\n  export PATH=\"%s:\$PATH\"\n\n" "$install_dir"
            ;;
    esac

    printf "\n"
    "$install_path" version
}

main
