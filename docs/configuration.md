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
