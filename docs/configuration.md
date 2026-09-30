# User configuration (`config.toml`)

jig looks for a config file at `~/.config/jig/config.toml`. All fields are
optional. If the file does not exist, jig falls back to sensible defaults.

```toml
forge_username = "jdoe"
author         = "John Doe"
license        = "Apache-2.0"
forge_token    = "your-forge-token"
template_dir   = "/path/to/templates"

# Automatically trust unknown ssh host keys when fetching remote templates
# (changed keys always fail). See "Remote template repositories" in the
# custom templates guide.
ssh_accept_new = false

# Optionally run bundle-backed commands through a container instead of the
# host's bundler. See "Running through voxbox".
[runner]
type   = "local"                            # "local" (default) or "voxbox"
engine = "docker"                           # "docker" (default) or "podman"
image  = "ghcr.io/voxpupuli/voxbox:latest"

# Defaults for template variables when a module is created or converted.
# See "Template variables" below.
[template.vars.default]
enable_junit_reporting = true
```

## Fields

| Field | Used by | Description |
|-------|---------|-------------|
| `forge_username` | [`jig new module`](commands/new.md), [`jig build`](commands/build.md) | Default Forge username for the module interview and package naming |
| `author` | [`jig new module`](commands/new.md) | Default author name for the module interview |
| `license` | [`jig new module`](commands/new.md) | Default license (falls back to `Apache-2.0`) |
| `forge_token` | [`jig release`](commands/release.md) | Forge API token used for publishing |
| `template_dir` | scaffolding commands | Path to a [custom template directory](custom-templates.md) |
| `ssh_accept_new` | remote template fetches | Trust unknown ssh host keys automatically; see [host key verification](custom-templates.md#host-key-verification) |
| `[runner]` | `validate`, `test`, `msync` | Container runner settings; see [Running through voxbox](voxbox.md) |
| `[template.vars]` | `new module`, `convert` | Defaults for [template variables](custom-templates.md#template-variables); see below |

## Template variables

`[template.vars]` holds your defaults for
[template variables](custom-templates.md#template-variables), in three
levels, most specific first:

```toml
[template.vars.default]            # every module
enable_junit_reporting = true

[template.vars.voxpupuli]          # modules by Forge user voxpupuli
enable_test_hiera = true

[template.vars.voxpupuli.nftables] # the module voxpupuli-nftables
beaker_fixture_modules = ["puppetlabs/concat"]
```

These are read only when `jig new module` or `jig convert` first records
a module's values in its [`jig.toml`](jig-toml.md#templatevars). After
that the module's `jig.toml` wins: editing your config has no effect on
existing modules, and `jig renew` never reads it. Precedence: a
`--template-var` flag, then values already in `jig.toml`, then the module
section, the author section, `default`, and finally the template's own
default. Variables the template does not declare are ignored, since your
config applies to every template you use.

Rules, each checked with an error naming the offending key:

- Variable names must be lowercase snake_case (`^[a-z][a-z0-9_]*$`).
- Values must be strings, booleans, numbers, or lists of those. A table
  under an author is always a module section, so variables cannot be
  tables.
- `default` always means the defaults section, never a Forge user.
- Variables cannot be set through environment variables.

## Overriding the config location

The config path can be overridden with the `--config` flag or the
`JIG_CONFIG` environment variable.

## Environment variables

Individual fields can also be set through `JIG_`-prefixed environment
variables (e.g. `JIG_FORGE_USERNAME`, `JIG_TEMPLATE_DIR`,
`JIG_RUNNER_TYPE`), which take precedence over the config file.

## What does *not* belong here

Settings that belong to a module rather than to a user — the template
repository it was scaffolded from, the renew allowlist, build packaging
rules — live in the module's [`jig.toml`](jig-toml.md) instead, so they
can be committed and shared with everyone working on the module.
