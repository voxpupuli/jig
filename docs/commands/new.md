# `jig new`

Scaffolds a new Puppet module, or generates components (classes, defined
types, facts, functions, providers, tasks, tests, transports) inside an
existing module.

## `jig new module`

Scaffolds a new Puppet module with the standard directory structure and
metadata.

```
jig new module <name> [flags]
```

jig will walk you through an interactive interview to collect module
metadata. Values from your [config file](../configuration.md) are used as
defaults. If no config is present, jig falls back to your system username
and full name.

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

**Module naming:** jig validates module names against Puppet's naming
conventions. Violations produce a warning but do not stop scaffolding.

Alongside the module files, `jig new module` writes two files that jig
itself owns: `metadata.json` and [`jig.toml`](../jig-toml.md).

## `jig new class`

Generates a new Puppet class manifest and its rspec-puppet spec file inside
the current module directory.

```
jig new class <name>
```

The class name follows standard Puppet naming conventions. Namespaced names
like `foo::bar` are supported and will generate the correct directory
structure under `manifests/`. The module name prefix must not be included
in the name.

## `jig new defined_type`

Generates a new Puppet defined type manifest and its rspec-puppet spec file
inside the current module directory.

```
jig new defined_type <name>
```

Defined type names follow the same conventions as class names. Namespaced
names like `foo::bar` are supported and will generate the correct directory
structure under `manifests/`. The module name prefix must not be included
in the name.

## `jig new fact`

Generates a new custom Facter fact and its spec file inside the current
module directory.

```
jig new fact <name>
```

Fact names may not contain `::`. The generated fact is placed in
`lib/facter/<name>.rb` and its spec in `spec/unit/facter/<name>_spec.rb`.

## `jig new function`

Generates a new Puppet language function and its spec file inside the
current module directory.

```
jig new function <name>
```

Function names follow standard Puppet naming conventions. The module name
is automatically prepended to form the fully qualified function name
(`<module>::<name>`). The generated function is placed in
`functions/<name>.pp` and its spec in `spec/functions/<name>_spec.rb`.

## `jig new provider`

Generates a new Puppet resource type and provider using the
[Resource API](https://github.com/puppetlabs/puppet-resource_api), along
with spec files for both, inside the current module directory.

```
jig new provider <name>
```

Provider names must start with a lowercase letter and contain only
lowercase letters, numbers, and underscores (`[a-z][a-z0-9_]*`). Four files
are generated:

- `lib/puppet/type/<name>.rb` — the Resource API type definition
- `lib/puppet/provider/<name>/<name>.rb` — the Resource API simple provider
- `spec/unit/puppet/type/<name>_spec.rb` — spec file for the type
- `spec/unit/puppet/provider/<name>/<name>_spec.rb` — spec file for the provider

## `jig new task`

Generates a new Puppet task and its metadata file inside the current module
directory.

```
jig new task <name>
```

Task names must start with a lowercase letter and contain only lowercase
letters, numbers, and underscores (`[a-z][a-z0-9_]*`). The special name
`init` is valid and maps the task to the module itself. Namespaced names
using `::` are not valid for tasks.

## `jig new test`

Generates a unit test for an existing class or defined type inside the
current module directory.

```
jig new test <name>
```

jig looks up the named resource by finding its manifest under `manifests/`
and inspects the file to determine whether it contains a class or a defined
type. The spec file is written to `spec/classes/` for classes or
`spec/defines/` for defined types. The name follows the same conventions as
`jig new class` and `jig new defined_type` — namespaced names like
`foo::bar` are supported, and the module name prefix must not be included.

An error is returned if the manifest does not exist, if no matching class
or defined type declaration is found in the file, or if a spec file for the
named resource already exists.

## `jig new transport`

Generates a new Puppet
[Resource API](https://github.com/puppetlabs/puppet-resource_api)
transport and its associated files inside the current module directory.

```
jig new transport <name>
```

Transport names must start with a lowercase letter and contain only
lowercase letters, numbers, and underscores (`[a-z][a-z0-9_]*`). Snake case
names like `my_device` are valid and will be converted to PascalCase
(`MyDevice`) where required by Ruby class naming conventions. Five files
are generated:

- `lib/puppet/transport/<name>.rb` — the transport implementation
- `lib/puppet/transport/schema/<name>.rb` — the Resource API transport schema
- `lib/puppet/util/network_device/<name>/device.rb` — legacy device compatibility shim
- `spec/unit/puppet/transport/<name>_spec.rb` — spec file for the transport
- `spec/unit/puppet/transport/schema/<name>_spec.rb` — spec file for the schema

## Template source flags

The following flags are available on all `jig new` subcommands. See
[Custom templates](../custom-templates.md) for the full story.

| Flag | Description |
|------|-------------|
| `-t, --template-dir` | Path to a custom template directory |
| `--template-url` | Git URL of a template repository to clone and use |
| `--template-ref` | Branch, tag, or ref to use with `--template-url`. Defaults to the remote's default branch. |
| `--ssh-accept-new` | Automatically trust unknown ssh host keys (like OpenSSH's `StrictHostKeyChecking=accept-new`). A changed key still fails. |

`--template-dir` and `--template-url` are mutually exclusive, and
`--template-ref` requires `--template-url`.

When no flag is given, `jig new` inside an existing module uses the
template source recorded in the module's
[`jig.toml`](../jig-toml.md#template) — so the whole team scaffolds from
the same templates with no flags at all.
