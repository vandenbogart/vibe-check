# vibe-check

A transparent package registry proxy that blocks recently published pip/uv/npm packages to protect against supply chain attacks.

Packages published less than 7 days ago are blocked by default. Old packages pass through untouched. When a package is blocked, you get a clear message with a one-command bypass.

## Install

```bash
go install github.com/vandenbogart/vibe-check@latest
```

## Setup

```bash
vibe-check setup
```

This does everything:
- Installs a persistent background daemon (launchd on macOS, systemd on Linux)
- Configures pip, uv, and npm to route through the proxy
- Backs up any existing package manager configs to `~/.vibe-check/backups/`

After setup, use pip/uv/npm as usual. No workflow changes.

### Existing configs

`vibe-check setup` backs up your existing configs before modifying them:
- `~/.config/pip/pip.conf` → `~/.vibe-check/backups/pip.bak`
- `~/.config/uv/uv.toml` → `~/.vibe-check/backups/uv.bak`
- `~/.npmrc` → `~/.vibe-check/backups/npm.bak`

For npm, existing settings (auth tokens, `save-exact`, etc.) are preserved — only the `registry=` line is replaced. `vibe-check teardown` restores all configs from backup.

## Usage

Install packages as usual. Old packages work normally:

```bash
pip install flask        # published 2024 → allowed
npm install express      # published 2024 → allowed
```

New packages get blocked:

```
$ pip install sketchy-pkg
ERROR: 403 Client Error: Forbidden
BLOCKED: sketchy-pkg@1.0.0 published 2 days ago (minimum age: 7 days)
To override: vibe-check allow pypi sketchy-pkg 1.0.0
Then retry your install.
```

## Allowing blocked packages

Allow a package and all its transitive dependencies:

```bash
vibe-check allow npm @apify/actors-mcp-server 0.9.16
```

This resolves the full dependency tree via dry-run install and adds every package to the allowlist. Then retry your install.

To allow a single package without resolving deps:

```bash
vibe-check allow --exact npm some-pkg 1.0.0
```

## Configuration

Change the minimum age threshold:

```bash
vibe-check set-min-age 14d
```

Change log verbosity (trace shows full request/response details):

```bash
vibe-check set-log-level trace   # trace, debug, info
```

All changes take effect immediately on the running daemon — no restart needed.

### Setup flags

```
vibe-check setup [flags]
  --pypi-port int      Port for PyPI proxy (default 3141)
  --npm-port int       Port for npm proxy (default 3142)
  --min-age string     Minimum package age (default "7d")
  --data-dir string    State directory (default ~/.vibe-check/)
  --log-level string   Log level: trace, debug, info (default "info")
```

## Uninstall

```bash
vibe-check teardown
```

Stops the daemon, removes the background service, and restores all package manager configs from backup.

## How it works

```
pip/uv  ──► localhost:3141 ──► age check ──► pypi.org
npm     ──► localhost:3142 ──► age check ──► registry.npmjs.org
```

The proxy intercepts download requests (not metadata/search). For each download, it checks when the package version was published. If it's too new, it returns 403. Otherwise it proxies the request through transparently.

The proxy is fail-closed: if it can't determine a package's age (upstream unreachable, unparseable filename), it blocks the download.

## Data

All state lives in `~/.vibe-check/`:

```
~/.vibe-check/
  config.json       # ports, min-age
  allowlist.txt     # manually allowed packages
  vibe-check.sock   # daemon admin socket
  vibe-check.log    # logs (when running via launchd/systemd)
  backups/          # original package manager configs
```
