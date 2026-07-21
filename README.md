# jig

A Go-based reimplementation of the [Puppet Development Kit (PDK)](https://github.com/puppetlabs/pdk), built to be fast, self-contained, and free of Ruby runtime dependencies.

## Why jig?

PDK has been an essential tool for Puppet module authors for years. When
Perforce moved PDK to a closed-source model, it created a real problem for teams
and individuals who depend on open tooling for their workflows. On top of that,
PDK carries a heavy Ruby runtime footprint, which adds friction to CI
environments and developer machines alike.

jig aims to replace the parts of PDK that matter most: scaffolding new modules,
building module packages, and cutting releases. It ships as a single static
binary with no external runtime required.

## Features

- **Scaffolding:** `jig new` generates modules, classes, defined types, facts,
  functions, providers, tasks, tests, and transports. Use `--skip-interview` to
  scaffold non-interactively.
- **Templates:** override the built-in templates locally, pull templates from
  remote repositories, and inspect them with `jig templates dump` and
  `jig templates resolve`.
- **Module lifecycle:** `jig renew` refreshes a module against its templates,
  and `jig convert` brings existing modules under jig management.
- **Build & release:** `jig build` packages a module and `jig release`
  publishes it to the Puppet Forge.
- **Quality checks:** `jig validate` runs static checks and `jig test unit`
  runs unit tests.
- **Fleet management:** `jig msync` keeps many modules in sync.

## Installation

Requires Go 1.21 or later.
```bash
git clone https://github.com/voxpupuli/jig.git
cd jig
go build -o jig .
```

Move the resulting binary somewhere in your `$PATH`:
```bash
mv jig /usr/local/bin/
```

No other dependencies or runtimes needed.

## Quick start

```bash
# Scaffold a new module (interactive interview)
jig new module mymodule
cd mymodule

# Generate a class inside it
jig new class myclass

# Run checks and tests
jig validate
jig test unit

# Build and publish to the Puppet Forge
jig build
jig release --version 1.0.0
```

## Documentation

Full documentation lives in [docs/](docs/README.md):

- **[Commands](docs/README.md#commands)** — every subcommand:
  [`new`](docs/commands/new.md), [`renew`](docs/commands/renew.md),
  [`convert`](docs/commands/convert.md),
  [`templates`](docs/commands/templates.md),
  [`build`](docs/commands/build.md), [`release`](docs/commands/release.md),
  [`validate`](docs/commands/validate.md), [`test`](docs/commands/test.md),
  [`msync`](docs/commands/msync.md)
- **[User configuration (`config.toml`)](docs/configuration.md)** — Forge
  credentials, interview defaults, environment variables
- **[Per-module configuration (`jig.toml`)](docs/jig-toml.md)** — template
  source, renew allowlist, build packaging rules
- **[Custom templates](docs/custom-templates.md)** — overriding the embedded
  templates from a directory or a remote git repository
- **[Running through voxbox](docs/voxbox.md)** — running the Ruby-backed
  commands in a container instead of a host Ruby toolchain
- **[Upgrading to 2.0](docs/upgrading-to-2.0.md)** — what changed from jig
  1.x and how to migrate modules and custom template directories
- **[Gotchas and migration notes](docs/gotchas.md)** — surprising behaviors,
  and notes for users coming from jig 1.x or PDK

## Contributing

Contributions are welcome. The project is in early stages, so the best place to
start is by opening an issue to discuss what you want to work on before sending
a PR. See the [contributing guide](docs/contributing.md) for project layout,
testing conventions, git hooks, and design notes.

## NOTICE

Some default template files included in this project are derived from the
[pdk-templates](https://github.com/puppetlabs/pdk-templates) project,
copyright Puppet Labs, and are used under the terms of the
[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## License

See [LICENSE](LICENSE).
