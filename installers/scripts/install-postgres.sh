#!/usr/bin/env bash
# install-postgres.sh — Install PostgreSQL and pgvector on macOS or Linux.
# After running this script, run: make prep-database
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"

die() { echo "Error: $*" >&2; exit 1; }

install_pgvector_from_source() {
    local pg_config="$1"
    echo "==> Building pgvector from source"
    local tmp
    tmp="$(mktemp -d)"
    git clone --depth 1 https://github.com/pgvector/pgvector.git "${tmp}/pgvector"
    make -C "${tmp}/pgvector" PG_CONFIG="${pg_config}"
    sudo make -C "${tmp}/pgvector" install PG_CONFIG="${pg_config}"
    rm -rf "${tmp}"
}

install_mac() {
    command -v brew &>/dev/null || die "Homebrew is required. Install from https://brew.sh"

    echo "==> Installing PostgreSQL via Homebrew"
    brew install postgresql@17

    echo "==> Installing pgvector via Homebrew"
    brew install pgvector

    echo "==> Starting PostgreSQL service"
    brew services start postgresql@17

    local pg_bin
    pg_bin="$(brew --prefix)/opt/postgresql@17/bin"
    if [[ ":$PATH:" != *":${pg_bin}:"* ]]; then
        echo ""
        echo "Add PostgreSQL to your PATH — append to your shell profile (~/.zshrc or ~/.bash_profile):"
        echo "  export PATH=\"${pg_bin}:\$PATH\""
        echo ""
        echo "Then reload: source ~/.zshrc  (or open a new terminal)"
    fi
}

install_apt() {
    echo "==> Installing PostgreSQL via apt"
    sudo apt-get update -qq
    sudo apt-get install -y postgresql postgresql-contrib

    local pg_ver
    pg_ver="$(psql --version 2>/dev/null | grep -oP '\d+' | head -1)"
    echo "==> Installing pgvector for PostgreSQL ${pg_ver}"

    if apt-cache show "postgresql-${pg_ver}-pgvector" &>/dev/null; then
        sudo apt-get install -y "postgresql-${pg_ver}-pgvector"
    else
        local pg_config
        pg_config="$(find /usr -name pg_config 2>/dev/null | head -1)"
        [[ -n "$pg_config" ]] || die "pg_config not found — cannot build pgvector"
        sudo apt-get install -y "postgresql-server-dev-${pg_ver}" build-essential git
        install_pgvector_from_source "$pg_config"
    fi

    sudo systemctl enable postgresql
    sudo systemctl start postgresql
}

install_dnf() {
    echo "==> Installing PostgreSQL via dnf"
    sudo dnf install -y postgresql-server postgresql-contrib
    sudo postgresql-setup --initdb
    sudo systemctl enable postgresql
    sudo systemctl start postgresql

    local pg_config
    pg_config="$(find /usr -name pg_config 2>/dev/null | head -1)"
    [[ -n "$pg_config" ]] || die "pg_config not found — cannot build pgvector"
    sudo dnf install -y gcc make git redhat-rpm-config "postgresql-devel"
    install_pgvector_from_source "$pg_config"
}

case "$(uname -s)" in
    Darwin)
        install_mac
        ;;
    Linux)
        if command -v apt-get &>/dev/null; then
            install_apt
        elif command -v dnf &>/dev/null; then
            install_dnf
        elif command -v yum &>/dev/null; then
            die "yum is not supported. Install PostgreSQL manually then run: make prep-database"
        else
            die "Unsupported package manager. Install PostgreSQL manually then run: make prep-database"
        fi
        ;;
    *)
        die "Unsupported OS: $(uname -s)"
        ;;
esac

echo ""
echo "==> PostgreSQL installed and running."
echo ""
echo "Next step: run 'make prep-database' from ${REPO_ROOT}"
