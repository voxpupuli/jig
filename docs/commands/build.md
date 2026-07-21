# `jig build`

Builds a module package suitable for uploading to the Puppet Forge. The
package is written to `pkg/<forge-user>-<module>-<version>.tar.gz`
relative to the current directory.

```
jig build
```

Metadata validation runs before the build. Errors abort the build;
warnings are printed and execution continues.

## What gets packaged

By default only files the
[Puppet module specification](https://github.com/puppetlabs/puppet-specifications/pull/157)
allows in a published module are packaged (the same allowlist
[puppet-modulebuilder](https://github.com/puppetlabs/puppet-modulebuilder)
uses) — `manifests/`, `lib/`, `data/`, `metadata.json`, and so on.
Everything else, including development files like `Gemfile`, `spec/`, and
dotfiles, stays out without any configuration.

Ignore files (`.pdkignore`, `.pmtignore`, `.gitignore`) are **not**
consulted; the build warns about any leftover `.pdkignore`-style file and
suggests removing it (`.gitignore` is exempt — it belongs to git, not the
build).

## Adjusting the file selection

The `[build]` section of the module's [`jig.toml`](../jig-toml.md)
adjusts this. `action` is the default treatment for every file and
`exceptions` lists gitignore-style globs treated the opposite way:

```toml
[build]
# "deny" (the default): package nothing except the spec allowlist plus the
# exceptions. "allow": package everything except the exceptions.
action     = "deny"
exceptions = ["/mycustomfile.txt"]
```

With `action = "deny"` the exceptions extend the built-in spec allowlist —
useful to ship a file the spec does not know about. With
`action = "allow"` jig packages everything except the exceptions, which is
the old `.pdkignore`-style denylist workflow relocated into `jig.toml`.

In both modes `pkg/`, `.git/`, `jig.toml` itself, and `.gitkeep` markers
are never packaged.
