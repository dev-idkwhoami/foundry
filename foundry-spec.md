# Foundry — Laravel Feature Installer Spec

## Overview

A Wails v2 (Go + Svelte) desktop app that clones a custom Laravel repository and applies opinionated, configurable feature layers on top. The installer uses a single Git repo as both the styled starter kit and the feature registry.

Target platform: **Windows** (Wails is cross-platform; other platforms may be added later).

Prerequisites: **Git**, **Laravel Herd** (provides PHP, Composer, and site provisioning).

---

## App Directory

All Foundry data lives in `%APPDATA%/Foundry/`:

```
%APPDATA%/Foundry/
├── config.yml          # App configuration
├── logs/               # Log files
└── tmp/                # Temporary working directories (clones, etc.)
```

### `config.yml`

```yaml
repository: "https://github.com/dev-idkwhoami/foundry-starter"

setup:
  - composer install
  - npm install
  - php artisan key:generate
  - php artisan storage:link
  - php artisan migrate:fresh --seed

cleanup:
  - features
  - app/Console/Commands/Foundry

flux_composer_url: "https://composer.fluxui.dev"

recent_directories: []
```

- **`repository`** — Git URL of the repository to clone as the project base
- **`setup`** — global commands that run after all feature patches and their hooks, before cleanup
- **`cleanup`** — list of paths (relative to project root) to delete from the installed project during cleanup
- **`flux_license_key`** / **`flux_username`** / **`flux_composer_url`** — Flux UI Pro credentials for Composer `auth.json` generation
- **`recent_directories`** — last 5 used working directories (managed automatically)

---

## Core Concepts

### The Repository
A custom Laravel repository that is always cloned as the base. It contains:
- Opinionated defaults (Livewire MFC, no emojis, etc.)
- A `features/` directory at the root containing all feature manifests and patches
- An `app/Console/Commands/Foundry/` directory with DX artisan commands for creating features and generating diffs
- Both directories are deleted from the project during cleanup (configurable via `config.yml` → `cleanup`)

### Features
Self-contained units of functionality defined by a manifest, a set of patches, and a mappings file. Features are applied on top of the styled starter after cloning.

### Git Patches
Each feature ships one or more `.diff` files generated via `git diff main -- . ':(exclude)features/'`. Patches always apply against the repository's `main` branch. The `:(exclude)features/` pathspec ensures patches never include the features directory itself, keeping them clean regardless of which features have been merged to `main`.

---

## Repository Structure

```
your/styled-starter (GitHub, default branch: main)
├── app/
├── resources/
├── ... (standard Laravel structure)
└── features/
    ├── teams/
    │   ├── manifest.yaml
    │   ├── patch.diff
    │   └── mappings.yaml
    ├── magic-link/
    │   ├── manifest.yaml
    │   ├── patch.diff
    │   └── mappings.yaml
    └── tenancy/
        ├── manifest.yaml
        ├── patch.diff
        ├── middleware.diff
        └── mappings.yaml
```

---

## Feature Manifest (`manifest.yaml`)

### Full Example

```yaml
id: tenancy
name: Tenancy
description: Adds multi-tenancy via global and local scopes.

requires:
  - teams

incompatible:
  - single-tenant-billing

patches:
  - file: patch.diff
    mode: auto

instructions:
  - text: "Add to the web middleware group in bootstrap/app.php"
    copy: "\\App\\Http\\Middleware\\EnsureTenant::class,"
  - text: "Add the BelongsTo{{tenant_noun:title}} trait to any model that should be scoped to the current tenant."
    copy: "use BelongsTo{{tenant_noun:title}};"

config:
  - key: tenant_noun
    label: Tenant noun
    type: text
    default: Team
    placeholder: e.g. Organization
  - key: tenant_noun_plural
    label: Tenant noun (plural)
    type: text
    default: Teams
  - key: billing_driver
    label: Billing driver
    type: select
    default: stripe
    options:
      - value: stripe
        label: Stripe
      - value: paddle
        label: Paddle
      - value: none
        label: No billing

hooks:
  pre-clone:
    - echo "Preparing tenancy feature..."
  post-clone:
    - composer require some/tenancy-package
  pre-herd:
    - echo "Before Herd setup..."
  post-herd:
    - echo "Herd site linked"
  pre-patch:
    - php artisan vendor:publish --tag=tenancy-config
  post-patch:
    - php artisan migrate --path=database/migrations/{{tenant_noun:snake}}
  pre-install:
    - echo "Before global commands..."
  post-install:
    - php artisan cache:clear
  pre-cleanup:
    - npm run build
  post-cleanup:
    - echo "Tenancy feature fully installed"
```

### Field Reference

#### Top-level fields

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | yes | Unique identifier, matches the directory name under `features/` |
| `name` | string | yes | Human-readable display name |
| `description` | string | no | Short description shown in the feature selection UI |
| `requires` | string[] | no | IDs of features this feature depends on |
| `incompatible` | string[] | no | IDs of features that cannot coexist with this one |
| `patches` | Patch[] | no | List of diff files to apply |
| `instructions` | Instruction[] | no | Manual steps shown to the user after installation (Step 6) |
| `config` | ConfigField[] | no | User-configurable options (rendered as form fields in Step 3) |
| `hooks` | Hooks | no | Commands to run at specific points in the installation pipeline |

#### `patches[]`

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | string | yes | Path to the `.diff` file relative to the feature directory |
| `mode` | string | no | `auto` (default) or `manual` |
| `instruction` | string | no | Plain-language instruction shown to the user for `manual` patches |

#### `instructions[]`

| Field | Type | Required | Description |
|---|---|---|---|
| `text` | string | yes | Plain-language instruction shown to the user in the manual steps checklist |
| `copy` | string | no | Copyable code snippet displayed below the instruction. Clicking copies to clipboard |

Both `text` and `copy` support `{{key:transformer}}` token syntax, resolved against the feature's config values.

#### `config[]`

| Field | Type | Required | Description |
|---|---|---|---|
| `key` | string | yes | Unique key within the feature, used in mappings as `{{key}}` or `{{key:transformer}}` |
| `label` | string | yes | Display label in the config form |
| `type` | string | yes | `text` (renders an input) or `select` (renders a dropdown) |
| `default` | string | no | Pre-filled default value |
| `placeholder` | string | no | Placeholder text for `text` inputs |
| `options` | Option[] | no | Required for `type: select` — list of choices |

#### `config[].options[]`

| Field | Type | Required | Description |
|---|---|---|---|
| `value` | string | yes | The value stored when this option is selected |
| `label` | string | yes | Display label in the dropdown |

#### `hooks`

Every installation stage has a `pre-` and `post-` hook. Hooks that are **per-feature** run within each feature's iteration (in topo order). Hooks that are **global** run once, iterating all selected features in topo order.

| Field | Type | Scope | Description |
|---|---|---|---|
| `pre-clone` | string[] | global | Before the repository is cloned. Runs from the working directory (project dir does not exist yet). |
| `post-clone` | string[] | global | After clone. Use for installing packages the patches depend on. |
| `pre-herd` | string[] | global | Before Herd site linking and database setup. |
| `post-herd` | string[] | global | After Herd site is linked, database created, and `.env` configured. |
| `pre-patch` | string[] | per-feature | Immediately before this feature's patches are applied. |
| `post-patch` | string[] | per-feature | Immediately after this feature's patches are applied. |
| `pre-install` | string[] | global | Before the global `setup` commands from `config.yml` run. |
| `post-install` | string[] | global | After the global `setup` commands complete. |
| `pre-cleanup` | string[] | global | Before cleanup (file deletion) begins. |
| `post-cleanup` | string[] | global | After cleanup completes. Last thing that runs. |

All hook commands support `{{key:transformer}}` token syntax, resolved against the feature's own config values.

```yaml
hooks:
  post-patch:
    - php artisan make:model {{tenant_noun}}
    - php artisan migrate --path=database/migrations/{{tenant_noun:snake}}_table.php
```

With `tenant_noun` set to `Organization`, these resolve to:
- `php artisan make:model Organization`
- `php artisan migrate --path=database/migrations/organization_table.php`

> **Note:** Global commands from `config.yml` → `setup` are **not** token-resolved (they are not feature-scoped).

### Patch Modes
- **`auto`** (default) — applied via `git apply` without user interaction
- **`manual`** — skipped during auto-apply; surfaced to the user in Step 6 as a checklist with the `instruction` text and the raw diff

### Instructions vs Manual Patches

The `instructions` field is the preferred way to define manual steps. Use it for actions the user must perform that don't have an associated diff file (e.g., adding a trait to models, registering middleware).

Legacy `mode: manual` patches still work and appear in the same checklist, but new features should use `instructions` instead — no empty placeholder diff files needed.

### `requires`
Declares that this feature depends on another feature. Used to:
1. Enforce patch application order (topological sort)
2. Auto-select required features in the UI when a dependent feature is checked
3. Prevent deselecting a required feature while a dependent feature is still selected

### `incompatible`
Declares features that cannot be used alongside this one. When a feature is selected, all features listed in its `incompatible` array are **pre-disabled** in the UI. This is checked bidirectionally — if A declares B as incompatible, B is also incompatible with A.

---

## Mappings (`mappings.yaml`)

Defines token substitution rules applied **to the .diff files before patching**. The Go backend reads the `.diff` file content, performs find-replace on the diff text itself using the mappings, then applies the now-customized diff via `git apply`. This means the patch lands with the correct names already in place.

Mappings target specific lines in the `.diff` file by line number. This includes **diff header lines** (`diff --git`, `+++`) — so file paths in the diff are rewritten too. This means `git apply` creates files directly at the correct renamed paths with the correct content, in a single step.

```yaml
mappings:
  - config_key: tenant_noun
    targets:
      - line: 1
        from: Team
        to: "{{tenant_noun}}"
      - line: 4
        from: Team
        to: "{{tenant_noun}}"
      - line: 8
        from: Team
        to: "{{tenant_noun}}"
      - line: 12
        from: team
        to: "{{tenant_noun:snake}}"
```

For a diff like:
```diff
diff --git a/app/Models/Team.php b/app/Models/Team.php   ← line 1: "Team" → "Organization"
new file mode 100644                                      ← line 2: untouched
--- /dev/null                                             ← line 3: untouched
+++ b/app/Models/Team.php                                 ← line 4: "Team" → "Organization"
@@ -0,0 +1,8 @@                                          ← line 5: untouched
+<?php                                                    ← line 6: untouched
+                                                         ← line 7: untouched
+class Team extends Model                                 ← line 8: "Team" → "Organization"
+{
+    protected $table = 'teams';                           ← line 10: mapped via different config_key
+}
```

After mapping resolution, `git apply` sees `+++ b/app/Models/Organization.php` and creates the file at the right path.

### Transformer Syntax

Tokens use the format `{{config_key:transformer}}`. A bare `{{config_key}}` uses the raw value. Transformers can be chained: `{{config_key:plural:lower}}`.

Available transformers (defined by the Go backend):

| Transformer | Example input | Output |
|---|---|---|
| `lower` | "Organization" | "organization" |
| `title` | "organization" | "Organization" |
| `plural` | "Organization" | "Organizations" |
| `snake` | "OrganizationUnit" | "organization_unit" |
| `kebab` | "OrganizationUnit" | "organization-unit" |
| `camel` | "organization unit" | "organizationUnit" |
| `dot` | "OrganizationUnit" | "organization.unit" |

Substitution resolves the config value through the transformer chain (left to right), then performs the replacement in the diff file content at the matching location.

---

## Conflict Detection

### Pre-disabled (manifest-based)
When a feature is selected, any feature listed in its `incompatible` array is immediately disabled in the UI. This is static and requires no computation.

### Dynamic (patch-based)
Patch compatibility is validated **at feature selection time** using `git apply --check` (dry-run) against a **cached clone** of the repository in `%APPDATA%/Foundry/tmp/<sha1(repoURL)>/`.

When a user toggles a feature on:
1. The installer runs `git apply --check` for the new feature's auto patches against the current selection state in the temp clone
2. If the check fails, the feature is **deselected and disabled** in the UI
3. The reason is logged (visible in `--verbose` mode)

This prevents the user from ever reaching the install step with an incompatible feature set.

---

## Installer Wizard — Steps

### Step 0: Startup (invisible to user)
- Clone (or `git pull` if cached) the repository into `%APPDATA%/Foundry/tmp/<sha1(repoURL)>/`
- Parse all `features/*/manifest.yaml` files to build the feature registry
- Build the incompatibility graph from `incompatible` declarations
- Frontend listens for `ready` event; polls `GetStartupResult()` as fallback if event was missed

### Step 1: Project Setup
- Project name
- Target directory (directory picker)
- Target directory warning if path already exists and is non-empty
- Environment checks (blocks advancement if critical prereqs missing):
  - **Git** — version detection, errors if missing
  - **Herd** — version detection, errors if missing
  - **Flux UI Pro** — checks if both username and license key are configured in Settings

### Step 2: Feature Selection
- List of available features read from the temp clone's `features/` directory
- Search input to filter features by name or description
- Scrollable card grid with custom scrollbar when features overflow the viewport
- Checkboxes with name, description
- Selecting a feature **auto-selects** its `requires` dependencies
- A required feature **cannot be deselected** while a dependent feature is still selected
- Incompatible features are **pre-disabled** based on manifest declarations
- Additional incompatibilities are dynamically **disabled** based on `git apply --check` dry-runs against the temp clone
- Features with only manual patches are visually distinguished

### Step 3: Feature Configuration
- One accordion section per selected feature
- Renders config fields from `manifest.yaml` → `config`
- Defaults pre-filled
- Live transformer preview only shown in `--debug` mode (developer tool)

### Step 4: Review
- Summary of selected features
- List of auto patches to be applied
- List of manual steps required after install
- "Install" CTA

### Step 5: Installation
Triggered on mount of the InstallProgress component. The backend runs the pipeline asynchronously, emitting structured events for real-time UI updates.

**Installation Pipeline — Execution Order:**

```
 ┌─────────────────────────────────────────────────────────┐
 │  hooks.pre-clone          (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  CLONE                                                  │
 │  Clone repository → <targetDir>/<projectName>/          │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.post-clone         (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.pre-herd           (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  HERD SETUP                                             │
 │  herd link --secure <sitename>                          │
 │  Copy .env.example → .env (if .env missing)             │
 │  CREATE DATABASE <dbname> (PostgreSQL via postgres db)   │
 │  Configure .env (DB creds + APP_URL)                    │
 │  Write auth.json (Flux credentials, if configured)      │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.post-herd          (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  PATCHING (per feature, in topological order)           │
 │  ┌───────────────────────────────────────────────┐      │
 │  │  hooks.pre-patch       (this feature)         │      │
 │  │  Apply auto patches    (resolve mappings      │      │
 │  │                         → git apply via stdin) │      │
 │  │  Collect manual patches + instructions          │      │
 │  │  (for Step 6, resolve tokens in both)          │      │
 │  │  hooks.post-patch      (this feature)         │      │
 │  └───────────────────────────────────────────────┘      │
 │  ... repeat for each selected feature ...               │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.pre-install        (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  GLOBAL SETUP                                           │
 │  Commands from config.yml → setup                       │
 │  (composer install, npm install, migrations, etc.)      │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.post-install       (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.pre-cleanup        (all features, topo order)    │
 ├─────────────────────────────────────────────────────────┤
 │  CLEANUP                                                │
 │  Delete paths from config.yml → cleanup                 │
 │  Reset cached clone (git checkout .)                    │
 ├─────────────────────────────────────────────────────────┤
 │  hooks.post-cleanup       (all features, topo order)    │
 └─────────────────────────────────────────────────────────┘
```

All feature hooks support `{{key:transformer}}` token resolution. Global `setup` commands are not token-resolved.

Each stage emits an ASCII banner line to the log panel (e.g. `── Clone ──────────────────────`) for visual separation.

**Event protocol:**
- `install:log` — `{ message, level }` — individual log lines
- `install:progress` — `{ stage, status }` — stage transitions (`running` / `done` / `error`)
- `install:error` — `{ stage, message }` — failure details
- `install:complete` — pipeline finished successfully
- `install:result` — `{ success, manualSteps, errorMessage, errorStage }` — final result

**UI:** Two-column layout with a stage sidebar (status icons per stage) and a terminal-style log panel with auto-scroll and custom scrollbars (both axes). The log panel stretches to fill available window height. On success: "Open in Explorer" button (opens target dir + quits) and "Close" button (quits). On error: error card with stage name and message.

### Step 6: Manual Steps Checklist
- One card per manual step (from `instructions` and legacy `mode: manual` patches) with feature badge, instruction text, and checkbox
- Instructions with a `copy` field show a monospace code block with a click-to-copy button
- Checked items show strikethrough and reduced opacity
- Progress counter ("X of Y steps completed")
- "Open Project in Explorer" button at top
- "Done" button disabled until all items are checked (opens explorer + quits)
- Empty state: "No manual steps required" with immediate close/open buttons

---

## Logging

Two modes controlled by a `--verbose` CLI flag or `FOUNDRY_VERBOSE=1` environment variable:

| Mode | What is logged |
|---|---|
| **Default** | Critical events only: command failures, patch apply errors, fatal exceptions |
| **Verbose** (`--verbose` or `FOUNDRY_VERBOSE=1`) | All operational detail: selected feature values, config resolutions, why features were disabled/incompatible, subprocess stdout/stderr, file operations |

Log files are per-day (`2026-04-09.log`), append mode. Multiple app starts on the same day write to the same file. Old logs are pruned at 16 files.

Logs are displayed in the Step 5 log panel and written to `%APPDATA%/Foundry/logs/`.

## Debug Mode

The `--debug` CLI flag enables developer-facing UI features:
- Transformer preview on the Feature Configuration step (shows all transform variants of config values)

---

## Authoring Workflow (Feature Development)

### Branch Strategy

The `main` branch is the **clean base** — it only contains:
- The base Laravel application
- The `features/` directory (manifests, patches, mappings)
- The `app/Console/Commands/Foundry/` DX artisan commands

**Feature code never lands on `main`.** Each feature lives on its own branch. Only the `features/<id>/` directory is published back to `main`.

### Development Flow

1. Create a feature branch: `git checkout -b features/magic-link`
2. Write code as normal against the starter
3. Generate patch: `git diff main -- . ':(exclude)features/' > features/magic-link/patch.diff`
4. Write `manifest.yaml` (and `mappings.yaml` if needed) in `features/magic-link/`
5. Commit everything on the feature branch (code + `features/` folder)
6. Publish only the `features/<id>/` directory to `main`:

```bash
# From the feature branch, after committing:
git stash
git checkout main
git checkout features/magic-link -- features/magic-link/
git add features/magic-link/
git commit -m "Publish magic-link feature"
git checkout features/magic-link
git stash pop
```

The feature branch stays around as the source of truth for the code. The patch file in `features/` is the portable artifact that Foundry applies at install time.

### Updating a Feature

When you change feature code:
1. Make changes on the feature branch
2. Re-generate the patch: `git diff main -- . ':(exclude)features/' > features/magic-link/patch.diff`
3. Commit on the feature branch
4. Re-publish `features/magic-link/` to `main` (same checkout flow as above)

### DX Artisan Commands

The starter repo includes artisan commands in `app/Console/Commands/Foundry/` that automate this workflow:
- **`foundry:diff`** — generates the patch diff for the current feature branch
- **`foundry:publish`** — commits current changes, publishes `features/<id>/` to `main`, and switches back

These commands are deleted from the project during cleanup (configured in `config.yml` → `cleanup`).

### Manual Steps

For steps the user must perform manually after installation (e.g. adding a trait to models, registering middleware), use the `instructions` field:

```yaml
instructions:
  - text: "Add to the web middleware group in bootstrap/app.php"
    copy: "\\App\\Http\\Middleware\\EnsureTenant::class,"
```

Legacy `mode: manual` patches with a separate diff file are still supported but `instructions` is preferred for new features.

---

## Wails App Architecture (Wails v2, Go 1.23, SvelteKit 2, Svelte 5)

### Go Backend

| Package | Purpose |
|---|---|
| `backend/appdata` | Bootstrap `%APPDATA%/Foundry/` directory structure, default `config.yml` |
| `backend/config` | YAML config load/save (`AppConfig` struct) |
| `backend/logger` | Structured logging (default/verbose), per-day log files (append mode), auto-pruning at 16 files, file + Wails event emission |
| `backend/git` | Cached clone-or-pull to `%APPDATA%/Foundry/tmp/<sha1>/`, clone to target directory |
| `backend/features` | Feature manifest/mappings parsing, registry with dependency/incompatibility graphs, topological sort, mapping resolver |
| `backend/transformer` | Token transformers: lower, title, plural, snake, camel, dot (chainable) |
| `backend/herd` | Herd site linking, `.env.example` → `.env` copy, PostgreSQL database creation (connects via `postgres` db), `.env` configuration |
| `backend/installer` | Installation orchestrator, patch applier, post-install command runner, cleanup |
| `backend/db` | SQLite database (via `modernc.org/sqlite`, pure Go) for tracking installations. Path-based lookup uses SHA1 hash of normalized path for case-insensitive matching on Windows |

**Key bound methods on `App` struct:**
- `GetConfig`, `GetStartupContext`, `GetStartupResult`, `GetFeatures`, `GetFeatureRegistry`
- `GetGitVersion`, `GetHerdVersion`, `GetPHPVersion`, `GetComposerVersion`
- `GetFluxLicenseKey`, `SetFluxLicenseKey`, `GetFluxUsername`, `SetFluxUsername`, `WriteAuthJSON`
- `SetRepository`, `SelectDirectory`, `AddRecentDirectory`, `GetRecentDirectories`
- `CheckPatchCompatibility` — `git apply --check` dry-run against cached clone
- `CheckTargetDirectory` — returns `"empty"` or `"not-empty"` for target path validation
- `ResolveToken` — apply transformer chain (used for live preview in `--debug` mode)
- `IsDebug` — returns whether `--debug` flag is set
- `OpenInExplorer` — opens a directory in Windows Explorer
- `OpenFileInEditor` — opens a file in the system default editor (`cmd /c start`)
- `Install` — triggers the full async installation pipeline
- `ListProjects` — returns all tracked installations from SQLite
- `HerdUnlink` — runs `herd unlink` in a project directory
- `ForgetProject` — unlinks from Herd (best-effort) and deletes the installation record
- `Quit` — closes the application

### Svelte Frontend
- **Framework:** SvelteKit 2 + Svelte 5 (runes), Tailwind CSS 4, neo-brutalist theme (tweakcn), Lucide icons, shadcn components
- **Wizard state machine** — steps: loading → dir-select → project-setup → features → config → review → install → manual
- **Stores:** `project.ts` (name, dir, target path), `features.ts` (registry, selection, config values, compat state), `wizard.ts` (step navigation, validation gates), `install.ts` (manual steps data, prerequisitesMet gate)
- **Step 0 (Startup):** Loading spinner while clone/pull + registry build completes. Listens for `ready` event with `GetStartupResult()` fallback for race condition
- **Step 1 (Project Setup):** Project name, directory display, target path warning, environment cards (Git, Herd, Flux Pro status). Blocks advancement if Git or Herd missing
- **Step 2 (Feature Selection):** Search input with filter, scrollable checkbox grid with custom scrollbar, auto-select dependencies, locked deps, static + dynamic disable, patch compat checking
- **Step 3 (Feature Config):** Accordion per feature, text/select inputs. Transformer preview only in `--debug` mode
- **Step 4 (Review):** Summary cards for project info, selected features, patches, config values, manual step count, Install CTA
- **Step 5 (Install Progress):** Stage sidebar with status icons, terminal-style log panel with auto-scroll and custom scrollbars, full-width responsive layout. ASCII stage banners in log. Success: "Open in Explorer" + "Close" (both quit app). Error: prominent error card with stage-specific guidance (patching, clone, herd), scrollable error detail
- **Step 6 (Manual Checklist):** Cards per manual step (instructions + legacy manual patches) with checkbox, feature badge, instruction text, optional copyable code snippet. Progress tracking, completion gating, "Open Project in Explorer" button. Empty state auto-enables close
- **Settings page:** Modal overlay for repository URL, Flux username + license key. Closing refreshes env cards
- **Project manager:** Modal overlay (header icon) listing tracked installations with Unlink Herd and Forget Project actions
- **UX:** Frameless window with custom title bar, text selection disabled outside inputs, no nav/footer on install/manual steps

### Network calls
The app pulls/clones the repository at startup (cached in `tmp/<sha1>/`) and clones to the target directory at install time. All subsequent operations (feature reading, patching, mapping) happen on disk.

---

## Installation Tracking

Successful installations are recorded in a SQLite database at `%APPDATA%/Foundry/foundry.db`.

### Schema

```sql
CREATE TABLE installations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    path_hash     TEXT    NOT NULL UNIQUE,  -- SHA1 of lowercased, cleaned project path
    project_path  TEXT    NOT NULL,
    project_name  TEXT    NOT NULL,
    repository    TEXT    NOT NULL,
    site_name     TEXT    NOT NULL,
    db_name       TEXT    NOT NULL,
    installed_at  TEXT    NOT NULL,         -- RFC 3339
    updated_at    TEXT    NOT NULL          -- RFC 3339
);

CREATE TABLE installed_features (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    installation_id INTEGER NOT NULL REFERENCES installations(id) ON DELETE CASCADE,
    feature_id      TEXT    NOT NULL,
    feature_name    TEXT    NOT NULL,
    config_values   TEXT    NOT NULL DEFAULT '{}',  -- JSON object
    installed_at    TEXT    NOT NULL                 -- RFC 3339
);
```

- Path hashing enables case-insensitive matching on Windows (`B:\Projects` = `b:\projects`)
- Upsert on `path_hash` — re-installing to the same directory updates rather than duplicates
- Per-feature config values stored as JSON for future management UI
- Database opened on startup (`db.Open()`), closed on shutdown (`defer db.Close()`)
- Recording happens after cleanup but before `install:complete` event

---

## Out of Scope (for now)

- Third-party feature contributions
- Feature versioning / base version pinning
- GUI for authoring features
- Non-Windows platforms
- Bundling Git/PHP/Herd installers
