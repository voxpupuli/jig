# jig documentation

jig is a Go-based reimplementation of the
[Puppet Development Kit (PDK)](https://github.com/puppetlabs/pdk): scaffolding
new modules, building module packages, and cutting releases, shipped as a
single static binary with no Ruby runtime required.

## Getting started

- [Installation](installation.md)

## Commands

| Command | Description |
|---------|-------------|
| [`jig new`](commands/new.md) | Scaffold a new module, or generate classes, defined types, facts, functions, providers, tasks, tests, and transports inside one |
| [`jig renew`](commands/renew.md) | Re-render allowlisted module files from the latest templates |
| [`jig convert`](commands/convert.md) | Overwrite `Gemfile`, `Rakefile`, and `spec/spec_helper.rb` in an existing module to work with voxbox and Vox Pupuli tooling |
| [`jig templates`](commands/templates.md) | Dump the embedded templates to disk, or debug where a template name resolves from |
| [`jig build`](commands/build.md) | Build a module package for the Puppet Forge |
| [`jig release`](commands/release.md) | Validate, version, build, and publish a module release to the Forge |
| [`jig validate`](commands/validate.md) | Run syntax, puppet-lint, and rubocop checks |
| [`jig test`](commands/test.md) | Run the module's unit tests |
| [`jig msync`](commands/msync.md) | Run msync through bundle (e.g. `jig msync update`) |

**Global flags** available on every command:

| Flag | Description |
|------|-------------|
| `--config` | Path to config file (default `~/.config/jig/config.toml`) |
| `--debug` | Enable debug output |

## Configuration

- [User configuration (`config.toml`)](configuration.md) — per-user settings:
  Forge credentials, defaults for the module interview, template location,
  and the container runner.
- [Per-module configuration (`jig.toml`)](jig-toml.md) — settings committed
  with the module: the template source it was scaffolded from, the `renew`
  allowlist, and build packaging rules.

## Guides

- [Custom templates](custom-templates.md) — overriding the embedded
  templates with your own directory or a remote git repository.
- [Running through voxbox](voxbox.md) — running the Ruby-backed commands
  (`validate`, `test unit`, `msync`) in a container instead of a host Ruby
  toolchain.
- [Upgrading to 2.0](upgrading-to-2.0.md) — what changed from jig 1.x
  and how to migrate modules and custom template directories.
- [Gotchas and migration notes](gotchas.md) — surprising behaviors worth
  knowing about, and notes for users coming from jig 1.x or PDK.

## Contributing

- [Contributing guide](contributing.md) — project layout, testing
  conventions, git hooks, and design notes.
