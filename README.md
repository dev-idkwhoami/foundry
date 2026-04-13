<p align="center">
  <img src=".github/assets/logo.svg" width="128" alt="Foundry" />
</p>

<h1 align="center">Foundry</h1>

<p align="center">
  A desktop installer for opinionated Laravel projects.<br/>
  Clone a starter repo, select features, configure, install — done.
</p>

---

## What it does

Foundry is a Wails v2 desktop app (Go + Svelte 5) that takes a custom Laravel repository and applies configurable feature layers on top via git patches. It handles the full setup pipeline: cloning, Herd site provisioning, database creation, patching, and post-install commands.

## Prerequisites

- **Git** — [git-scm.com](https://git-scm.com)
- **Laravel Herd** — [herd.laravel.com](https://herd.laravel.com)

Herd provides PHP, Composer, and local site management.

## Installation

Download the latest `foundry-setup.exe` from [Releases](../../releases) and run it. Foundry installs to `%LOCALAPPDATA%\Foundry`.

Alternatively, download `foundry-windows-amd64.zip` for a portable build.

## Usage

1. Launch Foundry from the Start Menu or run `foundry.exe` from a terminal
2. **Project Setup** — name your project and verify Git + Herd are available
3. **Features** — select which features to include (dependencies auto-resolve)
4. **Configuration** — configure feature options (model names, drivers, etc.)
5. **Review** — confirm selections and start the install
6. **Install** — watch the pipeline run in real-time
7. **Manual Steps** — complete any manual patches that couldn't be auto-applied

### CLI flags

| Flag | Description |
|---|---|
| `--verbose` | Enable verbose logging (or set `FOUNDRY_VERBOSE=1`) |
| `--debug` | Show developer tools (transformer preview in config step) |

### From a terminal

```bash
cd C:\Projects
foundry my-app
```

When launched from a directory, Foundry uses the current directory as the working directory and the argument as the project name.

### CLI tool (feature development)

A separate `foundry-cli` binary is included for feature development. Run from within the starter repository.

| Command | Description |
|---|---|
| `foundry-cli create [<name>]` | Scaffold a new feature (branch + directory + manifest + mappings) |
| `foundry-cli diff --feature <id>` | Generate a `.cdiff` from current branch changes |
| `foundry-cli publish --feature <id>` | Publish feature to its branch |
| `foundry-cli validate` | Check all features for patch conflicts |
| `foundry-cli check <id> --with f1,f2` | Check one feature against a set |

## Configuration

App config lives at `%APPDATA%\Foundry\config.yml`:

```yaml
repository: "https://github.com/your-org/your-starter"

setup:
  - composer install --no-interaction
  - npm install --no-fund --ignore-scripts
  - php artisan key:generate
  - php artisan storage:link
  - php artisan migrate:fresh --seed

cleanup: []  # optional commands to run after install (features/ is always deleted automatically)
```

Flux UI Pro credentials and other settings can be configured in the Settings panel (gear icon).

## Development

```bash
# Install dependencies
cd frontend && npm install && cd ..

# Run in dev mode
wails dev
```

Requires Go 1.25+, Node 20+, and the [Wails CLI](https://wails.io/docs/gettingstarted/installation).

## Building

```bash
wails build
go build -o build/bin/foundry-cli.exe ./cmd/cli/
```

Produces `build/bin/foundry.exe` and `build/bin/foundry-cli.exe`. For the NSIS installer, install NSIS and run:

```bash
makensis installer.nsi
```

## Spec

See [spec.md](foundry-spec.md) for the full technical specification.

## Roadmap

- [ ] **Post-install feature addition** — add command-only features after initial setup
- [ ] **Installation dashboard** — view project info, installed features, and config values

## License

Proprietary. All rights reserved.
