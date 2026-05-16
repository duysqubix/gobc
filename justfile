go            := env_var_or_default('GO', 'go')
pkgs          := './...'
bin_dir       := 'bin'
gobc_bin      := bin_dir / 'gobc'
cartdump_bin  := bin_dir / 'cartdump'
cover_out     := 'coverage.out'
cover_html    := 'coverage.html'
cover_min     := env_var_or_default('COVER_MIN', '0')

release_flags := '-trimpath -ldflags=-s -ldflags=-w'
debug_flags   := '-gcflags=all=-N -gcflags=all=-l'

default:
    @just --list --unsorted

# install Go (if missing) + Go dev tools (gopls, staticcheck, dlv, goimports)
bootstrap:
    #!/usr/bin/env bash
    set -euo pipefail
    GO_VERSION="${GO_VERSION:-1.26.3}"
    GO_INSTALL_DIR="${GO_INSTALL_DIR:-/usr/local}"

    echo "==> Checking Go..."
    if command -v go &> /dev/null; then
        echo "    Go already installed: $(go version)"
    else
        echo "    Go not found. Installing ${GO_VERSION} to ${GO_INSTALL_DIR}/go (sudo required)..."
        os=$(uname -s | tr '[:upper:]' '[:lower:]')
        case "$(uname -m)" in
            x86_64)         arch=amd64 ;;
            aarch64|arm64)  arch=arm64 ;;
            *) echo "ERROR: unsupported architecture $(uname -m)"; exit 1 ;;
        esac
        url="https://go.dev/dl/go${GO_VERSION}.${os}-${arch}.tar.gz"
        tmpfile=$(mktemp /tmp/go-XXXXXX.tar.gz)
        trap 'rm -f "$tmpfile"' EXIT
        echo "    downloading $url"
        curl -fsSL "$url" -o "$tmpfile"
        sudo rm -rf "${GO_INSTALL_DIR}/go"
        sudo tar -C "$GO_INSTALL_DIR" -xzf "$tmpfile"
        export PATH="${GO_INSTALL_DIR}/go/bin:$PATH"
        echo "    Go installed: $(go version)"
        echo
        echo "    *** Add to your shell rc (e.g. ~/.zshrc): ***"
        echo "        export PATH=${GO_INSTALL_DIR}/go/bin:\$HOME/go/bin:\$PATH"
        echo
    fi

    GO_BIN="$(go env GOPATH)/bin"
    export PATH="$GO_BIN:$PATH"

    echo
    echo "==> Installing dev tools to $GO_BIN..."
    tools=(
        "golang.org/x/tools/gopls@latest"
        "honnef.co/go/tools/cmd/staticcheck@latest"
        "github.com/go-delve/delve/cmd/dlv@latest"
        "golang.org/x/tools/cmd/goimports@latest"
    )
    for tool in "${tools[@]}"; do
        printf '    installing %-55s ... ' "$tool"
        if go install "$tool"; then echo ok; else echo FAIL; exit 1; fi
    done

    echo
    echo "==> Verifying..."
    missing=0
    for cmd in go gopls staticcheck dlv goimports; do
        if path=$(command -v "$cmd" 2>/dev/null); then
            printf '    %-14s %s\n' "$cmd" "$path"
        else
            printf '    %-14s NOT IN PATH\n' "$cmd"
            missing=$((missing + 1))
        fi
    done

    if [ $missing -gt 0 ]; then
        echo
        echo "*** $missing tool(s) not in PATH. Add this to your shell rc and re-source: ***"
        echo "    export PATH=\$HOME/go/bin:\$PATH"
        exit 1
    fi
    echo
    echo "Done. All dev tools available."

# vet + test + compile (default debug-symbols intact)
build: vet test compile

# vet + test + release compile (stripped, trimpath)
build-release: vet test (compile release_flags)

# vet + test + debug compile (no optimizations; dlv-friendly)
build-debug: vet test (compile debug_flags)

# compile binaries only (no tests) - fast iteration
compile flags='':
    @mkdir -p {{bin_dir}}
    {{go}} build {{flags}} -o {{gobc_bin}} ./cmd/gobc
    {{go}} build {{flags}} -o {{cartdump_bin}} ./cmd/cartdump

# compile gobc + run it; all args pass through (e.g. `just run --debug rom.gb`)
run *args: compile
    ./{{gobc_bin}} {{args}}

# go vet ./...
vet:
    {{go}} vet {{pkgs}}

# go test with race detector
test:
    {{go}} test -race {{pkgs}}

# run tests + write coverage profile + print total
test-cover:
    {{go}} test -race -covermode=atomic -coverprofile={{cover_out}} {{pkgs}}
    {{go}} tool cover -func={{cover_out}} | tail -1

# generate HTML coverage report
test-cover-html: test-cover
    {{go}} tool cover -html={{cover_out}} -o {{cover_html}}
    @echo "Coverage report written to {{cover_html}}"

# fail if total coverage < COVER_MIN env var (default 0)
test-cover-check: test-cover
    #!/usr/bin/env bash
    set -euo pipefail
    total=$({{go}} tool cover -func={{cover_out}} | tail -1 | awk '{print $3}' | sed 's/%//')
    if awk -v t="$total" -v m={{cover_min}} 'BEGIN { exit !(t+0 < m+0) }'; then
        printf 'FAIL: coverage %s%% < required %s%%\n' "$total" {{cover_min}}
        exit 1
    fi
    printf 'PASS: coverage %s%% >= required %s%%\n' "$total" {{cover_min}}

# run benchmarks
bench:
    {{go}} test -bench=. -benchmem -run='^$' {{pkgs}}

# go mod tidy
tidy:
    {{go}} mod tidy

# vet + staticcheck (if installed)
lint: vet
    @if command -v staticcheck >/dev/null 2>&1; then staticcheck {{pkgs}}; else echo "staticcheck not installed; skipping"; fi

# full CI pipeline (vet + coverage-gated tests + compile)
ci: vet test-cover-check compile

# install repo-tracked git pre-commit hook (gofmt + vet + staticcheck on staged .go files)
install-hooks:
    git config core.hooksPath .githooks
    chmod +x .githooks/*
    @echo "pre-commit hook active: runs gofmt + vet + staticcheck before each commit"

# remove the pre-commit hook (restores default .git/hooks/ behavior)
uninstall-hooks:
    git config --unset core.hooksPath
    @echo "pre-commit hook removed"

# remove bin/, coverage artifacts, test cache
clean:
    rm -rf {{bin_dir}} {{cover_out}} {{cover_html}} cartdump.txt
    {{go}} clean -testcache
