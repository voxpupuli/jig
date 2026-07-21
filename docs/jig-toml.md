# Per-module configuration (`jig.toml`)

Settings that belong to a module rather than to a user live in a
`jig.toml` in the module root, next to `metadata.json`. It is created by
[`jig new module`](commands/new.md) and committed to the module
repository, so everyone working on the module shares it. All sections are
optional; an absent section means jig's defaults.

```toml
# Template repository the module was scaffolded from; later jig commands
# in this module default to it.
[template]
url    = "ssh://git@my.git.server/jig_templates.git"
ref    = "main"
commit = "<commit the templates were fetched at>"

# Files `jig renew` may re-render and overwrite. Empty by default so
# nothing is overwritten accidentally.
[renew]
paths = []

# Which files go into the module package.
[build]
action     = "deny"
exceptions = []
```

## `[template]`

Records where the module's templates came from. Later `jig new`
invocations inside the module (for example `jig new class`) use the
recorded url and ref automatically, so the whole team scaffolds from the
same, current templates with no flags at all. An explicit `--template-dir`
or `--template-url` flag overrides the recorded values.

[`jig renew`](commands/renew.md) re-fetches the latest commit of the
recorded ref and, after a successful renew from a remote repository,
updates `commit` to the commit that was fetched.

See [Remote template repositories](custom-templates.md#remote-template-repositories)
for transports, authentication, and host key handling.

## `[renew]`

Gitignore-style globs (relative to the module root) selecting the files
[`jig renew`](commands/renew.md) may re-render and overwrite. Empty by
default, so nothing is overwritten until the module opts in.

## `[build]`

Adjusts which files [`jig build`](commands/build.md) packages. `action`
is the default treatment for every file (`"deny"` packages only the
Puppet module specification allowlist, `"allow"` packages everything) and
`exceptions` lists globs treated the opposite way. See the
[build documentation](commands/build.md#adjusting-the-file-selection) for
details.

## Trust decisions are never read from `jig.toml`

Settings like `ssh_accept_new` are deliberately read only from the
[user config](configuration.md), environment, or flags — never from
`jig.toml` or `metadata.json`: a cloned repository must not be able to
change security behavior for the people who clone it.
