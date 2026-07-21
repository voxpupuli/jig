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

## Status

jig is under active development. The table below reflects the current state of
planned functionality.

| Command            | Subcommand     | Status     |
|--------------------|----------------|------------|
| `new`              | `module`       | ✅ Working  |
| `new`              | `class`        | ✅ Working  |
| `new`              | `defined_type` | ✅ Working  |
| `new`              | `fact`         | ✅ Working  |
| `new`              | `function`     | ✅ Working  |
| `new`              | `provider`     | ✅ Working  |
| `new`              | `task`         | ✅ Working  |
| `new`              | `test`         | ✅ Working  |
| `new`              | `transport`    | ✅ Working  |
| `--skip-interview` |                | ✅ Working  |
| Template override  |                | ✅ Working  |
| Remote templates   |                | ✅ Working  |
| `templates`        | `dump`         | ✅ Working  |
| `templates`        | `resolve`      | ✅ Working  |
| `build`            |                | ✅ Working  |
| `release`          |                | ✅ Working  |
| `validate`         |                | ✅ Working  |
| `test`             | `unit`         | ✅ Working  |
| `update`           |                | ✅ Working  |

## Installation

### Build from source

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

## Usage

### `jig new module`

Scaffolds a new Puppet module with the standard directory structure and
metadata.
```
jig new module <n> [flags]
```

jig will walk you through an interactive interview to collect module metadata.
Values from your config file are used as defaults. If no config is present,
jig falls back to your system username and full name.

**Flags:**

| Flag | Description |
|------|-------------|
| `-u, --forge-user` | Your Puppet Forge username |
| `-a, --author` | Your full name |
| `-l, --license` | License type (default: from config, then `Apache-2.0`) |
| `-s, --summary` | One-line module summary |
| `-S, --source` | Source URL for the module |
| `-f, --force` | Overwrite an existing module directory. The existing directory is backed up with a timestamp before any files are written. |
| `-i, --skip-interview` | Skip the interactive interview and use flag values or defaults. |

### `jig new class`

Generates a new Puppet class manifest and its rspec-puppet spec file inside
the current module directory.
```
jig new class <n>
```

The class name follows standard Puppet naming conventions. Namespaced names
like `foo::bar` are supported and will generate the correct directory structure
under `manifests/`. The module name prefix must not be included in the name.

### `jig new defined_type`

Generates a new Puppet defined type manifest and its rspec-puppet spec file
inside the current module directory.
```
jig new defined_type <n>
```

Defined type names follow the same conventions as class names. Namespaced
names like `foo::bar` are supported and will generate the correct directory
structure under `manifests/`. The module name prefix must not be included in
the name.

### `jig new fact`

Generates a new custom Facter fact and its spec file inside the current module
directory.
```
jig new fact <n>
```

Fact names may not contain `::`. The generated fact is placed in
`lib/facter/<name>.rb` and its spec in `spec/unit/facter/<name>_spec.rb`.

### `jig new function`

Generates a new Puppet language function and its spec file inside the current
module directory.
```
jig new function <n>
```

Function names follow standard Puppet naming conventions. The module name is
automatically prepended to form the fully qualified function name
(`<module>::<name>`). The generated function is placed in
`functions/<name>.pp` and its spec in `spec/functions/<name>_spec.rb`.

### `jig new provider`

Generates a new Puppet resource type and provider using the
[Resource API](https://github.com/puppetlabs/puppet-resource_api), along with
spec files for both, inside the current module directory.
```
jig new provider <n>
```

Provider names must start with a lowercase letter and contain only lowercase
letters, numbers, and underscores (`[a-z][a-z0-9_]*`). Four files are
generated:

- `lib/puppet/type/<name>.rb` — the Resource API type definition
- `lib/puppet/provider/<name>/<name>.rb` — the Resource API simple provider
- `spec/unit/puppet/type/<name>_spec.rb` — spec file for the type
- `spec/unit/puppet/provider/<name>/<name>_spec.rb` — spec file for the provider

### `jig new task`

Generates a new Puppet task and its metadata file inside the current module
directory.
```
jig new task <n>
```

Task names must start with a lowercase letter and contain only lowercase
letters, numbers, and underscores (`[a-z][a-z0-9_]*`). The special name
`init` is valid and maps the task to the module itself. Namespaced names
using `::` are not valid for tasks.

### `jig new test`

Generates a unit test for an existing class or defined type inside the current
module directory.
```
jig new test <n>
```

jig looks up the named resource by finding its manifest under `manifests/` and
inspects the file to determine whether it contains a class or a defined type.
The spec file is written to `spec/classes/` for classes or `spec/defines/` for
defined types. The name follows the same conventions as `jig new class` and
`jig new defined_type` -- namespaced names like `foo::bar` are supported, and
the module name prefix must not be included.

An error is returned if the manifest does not exist, if no matching class or
defined type declaration is found in the file, or if a spec file for the named
resource already exists.

### `jig new transport`

Generates a new Puppet
[Resource API](https://github.com/puppetlabs/puppet-resource_api) transport
and its associated files inside the current module directory.
```
jig new transport <n>
```

Transport names must start with a lowercase letter and contain only lowercase
letters, numbers, and underscores (`[a-z][a-z0-9_]*`). Snake case names like
`my_device` are valid and will be converted to PascalCase (`MyDevice`) where
required by Ruby class naming conventions. Five files are generated:

- `lib/puppet/transport/<n>.rb` — the transport implementation
- `lib/puppet/transport/schema/<n>.rb` — the Resource API transport schema
- `lib/puppet/util/network_device/<n>/device.rb` — legacy device compatibility shim
- `spec/unit/puppet/transport/<n>_spec.rb` — spec file for the transport
- `spec/unit/puppet/transport/schema/<n>_spec.rb` — spec file for the schema

**Flags on `jig new`:**

The following flags are available on all `jig new` subcommands:

| Flag | Description |
|------|-------------|
| `-t, --template-dir` | Path to a custom template directory. See [Template Overrides](#template-overrides) below. |
| `--template-url` | Git URL of a template repository to clone and use. See [Remote template repositories](#remote-template-repositories) below. |
| `--template-ref` | Branch, tag, or ref to use with `--template-url`. Defaults to the remote's default branch. |
| `--ssh-accept-new` | Automatically trust unknown ssh host keys (like OpenSSH's `StrictHostKeyChecking=accept-new`). A changed key still fails. |

**Global flags:**

| Flag | Description |
|------|-------------|
| `--config` | Path to config file |
| `--debug` | Enable debug output |

**Module naming:** jig validates module names against Puppet's naming
conventions. Violations produce a warning but do not stop scaffolding.

### `jig templates dump`

Extracts all embedded default templates to a directory on disk. This is useful
as a starting point for creating your own custom templates. If the destination
directory already exists it will be renamed with a timestamp suffix before
writing.
```
jig templates dump <destination>
```

For example:
```bash
jig templates dump ~/.config/jig/templates
```

You can then edit the files in the destination directory and point jig at them
using `--template-dir` or the `template_dir` config key.

### `jig templates resolve`

Shows where a logical template name resolves from, for debugging custom
template setups. It reports which template source is in effect (flags, the
module's `jig.toml`, the config file, or the embedded defaults), every path
checked in order, and the winning file. The lookup is the same one the
scaffolding commands use, so the result is exactly what `jig new` would
render.
```
jig templates resolve <name>
```

For example, inside a module directory:
```console
$ jig templates resolve class/class.pp --template-dir ~/.config/jig/templates
external template directory: /home/me/.config/jig/templates (from --template-dir flag)
  looking for /home/me/.config/jig/templates/class/class.pp.tmpl (external) ... not found
  looking for /home/me/.config/jig/templates/class/class.pp (external) ... not found
not found in external template directory, falling back to embedded templates
  looking for templates/class/class.pp.tmpl (embedded) ... found
resolved class/class.pp to embedded template templates/class/class.pp.tmpl (rendered with text/template)
```

It accepts the same `--template-dir`, `--template-url`, `--template-ref`, and
`--ssh-accept-new` flags as `jig new`, and exits non-zero if the name does not
resolve in any source.

### `jig build`

Builds a module package suitable for uploading to the Puppet Forge. The
package is written to `pkg/<forge-user>-<module>-<version>.tar.gz` relative
to the current directory.
```
jig build
```

Metadata validation runs before the build. Errors abort the build; warnings
are printed and execution continues.

By default only files the
[Puppet module specification](https://github.com/puppetlabs/puppet-specifications/pull/157)
allows in a published module are packaged (the same allowlist
[puppet-modulebuilder](https://github.com/puppetlabs/puppet-modulebuilder) uses)
— `manifests/`, `lib/`, `data/`, `metadata.json`, and so on. Everything else,
including development files like `Gemfile`, `spec/`, and dotfiles, stays out
without any configuration. Ignore files (`.pdkignore`, `.pmtignore`,
`.gitignore`) are **not** consulted; the build warns about any leftover
`.pdkignore`-style file and suggests removing it (`.gitignore` is exempt —
it belongs to git, not the build).

The `[build]` section of the module's `jig.toml` adjusts this. `action` is the
default treatment for every file and `exceptions` lists gitignore-style globs
treated the opposite way:

```toml
[build]
# "deny" (the default): package nothing except the spec allowlist plus the
# exceptions. "allow": package everything except the exceptions.
action     = "deny"
exceptions = ["/mycustomfile.txt"]
```

With `action = "deny"` the exceptions extend the built-in spec allowlist —
useful to ship a file the spec does not know about. With `action = "allow"`
jig packages everything except the exceptions, which is the old
`.pdkignore`-style denylist workflow relocated into `jig.toml`. In both modes
`pkg/`, `.git/`, `jig.toml` itself, and `.gitkeep` markers are never packaged.

### `jig release`

Validates metadata, sets the version, builds the module package, and publishes
it to the Puppet Forge.
```
jig release [flags]
```

The release sequence is:

1. Validate the version string and module metadata (unless `--skip-validation`).
2. Write the new version into `metadata.json`.
3. Build the module package (unless `--skip-build`).
4. Upload the package to the Forge (unless `--skip-publish`).

A Forge API token is required for publishing. Set `forge_token` in your config
file or pass it via `--token`. You can generate a token from your account page
on the [Puppet Forge](https://forge.puppet.com).

**Flags:**

| Flag | Description |
|------|-------------|
| `-v, --version` | Version to release, e.g. `1.2.3` (required) |
| `-k, --token` | Forge API token (overrides `forge_token` in config) |
| `--skip-validation` | Skip metadata validation |
| `--skip-build` | Skip building the module archive |
| `--skip-publish` | Skip publishing to the Forge |

If `--skip-build` is set without `--skip-publish`, jig expects the archive to
already exist under `pkg/`. An error is returned if it is not found.

### `jig validate`

Runs validation checks against the current module. By default this mirrors
`pdk validate`: syntax (`rake validate`), puppet-lint (`rake lint`) and
rubocop (`rake rubocop`) all run, each as its own rake invocation. It shells
out to `bundle exec rake <task>` per check, so it requires a Ruby toolchain
and the module's bundled gems to be installed — or a container runner (see
[Running through voxbox](#running-through-voxbox)).
```
jig validate [flags] [-- args...]
```

| Flag | Description |
| ---- | ----------- |
| `-s`, `--syntax` | Run syntax checks (`rake validate`) |
| `-l`, `--lint` | Run puppet-lint checks (`rake lint`) |
| `-r`, `--rubocop` | Run rubocop checks (`rake rubocop`) |

With no flags all three checks run. Passing one or more flags runs only the
selected checks, e.g. `jig validate -s` for syntax only or `jig validate -sl`
to skip rubocop. Checks run in order and stop at the first failure,
propagating its exit code.

Any arguments after `--` are passed through verbatim to each underlying rake
invocation.

### `jig test unit`

Runs the module's unit tests. This is a passthrough command that shells out to
`bundle exec rake spec`, so it requires a Ruby toolchain and the module's
bundled gems to be installed — or a container runner (see
[Running through voxbox](#running-through-voxbox)).
```
jig test unit [args...]
```

Any additional arguments are passed through verbatim to the underlying rake
invocation.

### `jig update`

Synchronises the module's managed files from the module's templates. This is a
passthrough command that shells out to `bundle exec msync update`, so it
requires a Ruby toolchain and the module's bundled gems to be installed — or a
container runner (see [Running through voxbox](#running-through-voxbox)).
```
jig update [args...]
```

Any additional arguments are passed through verbatim to the underlying msync
invocation.

## Running through voxbox

The `validate`, `test unit`, and `update` commands run a Ruby toolchain under
the hood. By default that is the host's `bundle`, which needs a working
Ruby/bundler install. Instead, jig can run them inside the
[voxbox](https://github.com/voxpupuli/container-voxbox) container, so the only host
dependency is a container engine. This is especially handy on Windows, where a
system-wide bundler install is awkward.

Enable it via the `[runner]` section of your config file:
```toml
[runner]
type   = "voxbox"                        # "local" (default) or "voxbox"
engine = "docker"                        # container engine: "docker" (default) or "podman"
image  = "ghcr.io/voxpupuli/voxbox:latest"  # container image to run
```

With `type = "voxbox"`, your module root (the current directory) is mounted at
`/repo` inside the container and used as the working directory, so the toolchain
and gems come from the image rather than the host.

The voxbox image's entrypoint is already `bundle exec rake`, so the rake-based
commands pass their tasks straight through. `jig test unit` runs roughly:
```bash
docker run --rm -i -v "$PWD:/repo:Z" -w /repo \
  ghcr.io/voxpupuli/voxbox:latest spec
```

`jig update` uses `msync`, which is not a rake task, so it overrides the
entrypoint to run bundle directly:
```bash
docker run --rm -i -v "$PWD:/repo:Z" -w /repo --entrypoint bundle \
  ghcr.io/voxpupuli/voxbox:latest exec msync update
```

Each setting can also be supplied through the environment, which overrides the
config file:
```bash
export JIG_RUNNER_TYPE=voxbox
export JIG_RUNNER_ENGINE=podman
export JIG_RUNNER_IMAGE=ghcr.io/voxpupuli/voxbox:latest
```

## Template Overrides

jig embeds default templates for all generated files. If you want to customise
them, you can point jig at a directory of your own templates. Any template
found in your custom directory takes precedence over the embedded default.
Templates not present in your custom directory fall back to the embedded
defaults automatically, so you only need to include the files you want to
change.

The easiest way to get started is to run `jig templates dump` to extract the
default templates, then edit the ones you want to change.

### Rendered vs verbatim files

A file ending in `.tmpl` is rendered with Go's
[text/template](https://pkg.go.dev/text/template) and written with the suffix
stripped: `README.md.tmpl` becomes `README.md` in the generated module. Every
other file is copied to the output byte-for-byte. This means files that use
`{{ ... }}` for their own purposes -- GitHub Actions workflows, for example --
work without escaping as long as they don't carry the `.tmpl` suffix.

A file and its `.tmpl` variant side by side (`foo.yml` and `foo.yml.tmpl`)
would produce the same output file, so jig treats that as an error.

### The module template tree

The `module/` directory of the template tree mirrors the generated module
exactly: every file under `module/` is written to the same relative path in
the new module. There is no mapping in jig's source code, so your custom
template directory can add files jig knows nothing about -- drop
`module/.github/workflows/ci.yml` into your template directory and every
module you generate gets it. An empty directory is represented by a `.gitkeep`
file at that path.

Two paths are reserved and always generated by jig itself: `metadata.json`
and `jig.toml`. If your template tree contains them they are ignored with a
warning.

Overrides match on the *output* path: a verbatim `module/README.md` in your
custom directory replaces the embedded `module/README.md.tmpl`.

> **Migrating from jig 1.x:** template files used to be rendered
> unconditionally and some had special names (`gitignore` instead of
> `.gitignore`, `spec/init_spec.rb` instead of `spec/classes/init_spec.rb`).
> Rename rendered templates to add the `.tmpl` suffix and move files to their
> literal output paths, otherwise they are copied verbatim with `{{ ... }}`
> left unrendered.

### Template directory structure

Your custom template directory must mirror the structure of jig's embedded
templates (run `jig templates dump` to see the full tree):
```
templates/
  module/            # mirrors the generated module exactly
    .devcontainer/
      devcontainer.json
    .editorconfig
    .gitignore
    .overcommit.yml
    .rubocop.yml
    CHANGELOG.md
    Gemfile
    README.md.tmpl
    Rakefile.tmpl
    hiera.yaml
    data/
      .gitkeep
      common.yaml
    manifests/
      init.pp.tmpl
    spec/
      acceptance/
        init_spec.rb.tmpl
      classes/
        init_spec.rb.tmpl
      default_facts.yml
      spec_helper.rb
      spec_helper_acceptance.rb
  class/
    class.pp.tmpl
    class_spec.rb.tmpl
  type/
    defined_type.pp.tmpl
    defined_type_spec.rb.tmpl
  fact/
    fact.rb.tmpl
    fact_spec.rb.tmpl
  function/
    function.pp.tmpl
    function_spec.rb.tmpl
  provider/
    type.rb.tmpl
    type_spec.rb.tmpl
    provider.rb.tmpl
    provider_spec.rb.tmpl
  task/
    task.sh
    metadata.json
  transport/
    transport.rb.tmpl
    transport_spec.rb.tmpl
    device.rb.tmpl
    schema_transport.rb.tmpl
    schema_transport_spec.rb.tmpl
```

Unlike the `module/` tree, the component directories (`class/`, `fact/`,
`task/`, ...) use fixed file names: jig picks the file it needs and derives
the destination from the component name you pass on the command line.

### Configuring the template directory

There are three ways to tell jig where your custom templates live, in order
of precedence:

**Command line flag:**
```bash
jig new --template-dir /path/to/templates module mymodule
```

**Environment variable:**
```bash
export JIG_TEMPLATE_DIR=/path/to/templates
jig new module mymodule
```

**Config file** (`~/.config/jig/config.toml`):
```toml
template_dir = "/path/to/templates"
```

### Remote template repositories

Instead of a directory on disk, jig can fetch templates straight from a git
repository. This is the natural fit for teams: everyone shares one template
repo instead of keeping a checkout at the same local path.

```bash
jig new module --template-url 'ssh://git@my.git.server/jig_templates.git' --template-ref my_branch mymodule
```

jig makes a shallow clone into a temporary directory, uses it exactly like a
`--template-dir` (including per-file fallback to the embedded templates), and
deletes the clone afterwards. The repository layout is the same as a template
override directory.

The module's `jig.toml` records where the templates came from:

```toml
[template]
url    = "ssh://git@my.git.server/jig_templates.git"
ref    = "my_branch"
commit = "<commit the templates were fetched at>"
```

Later `jig new` invocations inside the module (for example `jig new class`)
use the recorded url and ref automatically, so the whole team scaffolds from
the same, current templates with no flags at all. An explicit
`--template-dir` or `--template-url` flag overrides the recorded values.
(Modules scaffolded by jig 1.x recorded these as `template-url` etc. in
`metadata.json`. Those keys are not supported: jig ignores them and warns,
with a suggestion to move them to the `[template]` section of `jig.toml` and
remove them from `metadata.json`.)

**Supported transports and authentication:**

- **ssh** — authenticated through your running ssh-agent (`SSH_AUTH_SOCK`).
  Keys with passphrases work as long as they are loaded in the agent.
- **http(s)** — anonymous access only; private repositories over https are
  not supported yet (use ssh for those).

**Host key verification:** ssh server keys are checked against
`~/.ssh/known_hosts` (created if missing, `$SSH_KNOWN_HOSTS` overrides the
path). On first contact with an unknown host, jig shows the key fingerprint
and asks before continuing, like OpenSSH does. In non-interactive contexts
(CI), pass `--ssh-accept-new`, set `ssh_accept_new = true` in the config, or
export `JIG_SSH_ACCEPT_NEW=true` to accept unknown hosts automatically — the
fingerprint is still printed for the log. A host key that *differs* from the
recorded one always fails, with no override; if a server legitimately rotated
its key, remove the stale entry (`ssh-keygen -R <host>`) and connect again.
Because it is a per-user trust decision, `ssh_accept_new` is read only from
the config, environment, or flag — never from `jig.toml` or `metadata.json`,
which are shared through the module repository.

## Configuration

jig looks for a config file at `~/.config/jig/config.toml`. All fields are
optional. If the file does not exist, jig falls back to sensible defaults.
```toml
forge_username = "jdoe"
author         = "John Doe"
license        = "Apache-2.0"
forge_token    = "your-forge-token"
template_dir   = "/path/to/templates"

# Automatically trust unknown ssh host keys when fetching remote templates
# (changed keys always fail). See "Remote template repositories" above.
ssh_accept_new = false

# Optionally run bundle-backed commands through a container instead of the
# host's bundler. See "Running through voxbox" above.
[runner]
type   = "local"                            # "local" (default) or "voxbox"
engine = "docker"                           # "docker" (default) or "podman"
image  = "ghcr.io/voxpupuli/voxbox:latest"
```

The config path can be overridden with the `--config` flag or the
`JIG_CONFIG` environment variable. Individual fields can also be set through
`JIG_`-prefixed environment variables (e.g. `JIG_FORGE_USERNAME`,
`JIG_RUNNER_TYPE`), which take precedence over the config file.

### Per-module configuration (`jig.toml`)

Settings that belong to a module rather than to a user live in a `jig.toml`
in the module root, next to `metadata.json`. It is created by
`jig new module` and committed to the module repository, so everyone working
on the module shares it. All sections are optional; an absent section means
jig's defaults.

```toml
# Template repository the module was scaffolded from; later jig commands in
# this module default to it. See "Remote template repositories" above.
[template]
url    = "ssh://git@my.git.server/jig_templates.git"
ref    = "main"
commit = "<commit the templates were fetched at>"

# Files the upcoming `jig renew` command may re-render and overwrite. Empty
# by default so nothing is overwritten accidentally.
[renew]
paths = []

# Which files go into the module package. See "jig build" above.
[build]
action     = "deny"
exceptions = []
```

Trust decisions (like `ssh_accept_new`) are deliberately never read from
`jig.toml`: a cloned repository must not be able to change security behavior
for the people who clone it.

## Contributing

Contributions are welcome. The project is in early stages, so the best place to
start is by opening an issue to discuss what you want to work on before sending
a PR.

### Project layout
```
.
├── main.go
├── commands/        # Cobra command definitions
└── internal/
    ├── build/
    ├── config/
    ├── forge/
    ├── module/      # Module metadata and validation
    ├── release/
    ├── scaffold/    # Scaffolding orchestration
    └── template/   # Template rendering with fallback logic
        └── templates/  # Embedded default templates
```

### Testing

Run the full test suite with:
```bash
go test ./...
```

Tests live alongside the source files they cover (`*_test.go`), which is
the standard Go convention. The `commands/` and `internal/config/` packages
do not currently have tests -- the former is thin Cobra wiring and the latter
is thin Viper wiring, so the internal packages are where the meaningful
coverage lives.

A few patterns used throughout the test suite that contributors should follow:

- **Table-driven tests** for functions with multiple input variations. Use a
  `cases := []struct{...}` slice and `t.Run` for each case.
- **`t.TempDir()`** for any test that touches the filesystem. It is cleaned up
  automatically after the test and requires no `defer os.Remove`.
- **`fakeRenderer`** in `internal/scaffold` implements the `scaffold.Renderer`
  interface and can be used to test template rendering paths without hitting
  the real embedded templates.
- **`makeBuildDir`** in `internal/build`, **`makeModuleDir`** in
  `internal/scaffold`, and **`makeModuleDir`** in `internal/release` are
  shared helpers that create realistic on-disk module structures for tests
  that need them. **`fakePublisher`** in `internal/release` implements the
  `forge.Publisher` interface for testing the release sequence without making
  real HTTP calls.
- Both characterization tests (pinning current behavior) and adversarial tests
  (checking rejection of invalid or malicious input) are expected. When adding
  a new feature, include both.

### Git hooks

A pre-commit hook is provided in `githooks/` that runs `gofmt`, `go vet`,
`go test ./...`, and `govulncheck ./...` before each commit. Enable it with:
```bash
git config core.hooksPath githooks
```

### Design notes for contributors

- Templates are embedded via `go:embed`. External templates take precedence
  over embedded ones, with per-file fallback to embedded defaults when a custom
  template is not found. Template names are validated to prevent path traversal
  before any file is read.
- `--force` never deletes existing files outright. It creates a timestamped
  backup of the target directory first.
- Module name validation uses a `ValidationResult` type with an iota-based
  `Severity`. Violations at the `Warning` level do not halt execution. Version
  strings must be valid semver (`MAJOR.MINOR.PATCH`). URL fields (`source`,
  `project_page`, `issues_url`) must use `http` or `https` schemes when
  present; invalid URLs are errors that abort the build and release.
- The Forge HTTP client (`internal/forge`) is hidden behind a `Publisher`
  interface so the release sequence can be tested without making real network
  requests.
- Component names (module names, class names, defined type names) are validated
  to reject empty strings, path separators, and traversal sequences before they
  are used to construct filesystem paths.
- `os.Getwd()` is called only in the `commands/` layer. Internal packages
  receive directory paths as arguments, which keeps them testable without
  manipulating the process working directory.
- Config is handled with [Viper](https://github.com/spf13/viper).

## NOTICE

Some default template files included in this project are derived from the
[pdk-templates](https://github.com/puppetlabs/pdk-templates) project,
copyright Puppet Labs, and are used under the terms of the
[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## License

See [LICENSE](LICENSE).