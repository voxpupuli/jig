# `jig msync`

Runs [modulesync](https://github.com/voxpupuli/modulesync) (`msync`)
through bundle. This is a passthrough command that shells out to
`bundle exec msync <args>`, so it requires a Ruby toolchain and the
module's bundled gems to be installed — or a container runner (see
[Running through voxbox](../voxbox.md)).

```
jig msync [command...]
```

The most common use is synchronising the module's managed files from the
module's templates:

```bash
jig msync update
```

All arguments are passed through verbatim to the underlying msync
invocation — jig does no flag parsing of its own here.

> **Note:** in jig 1.x this command was `jig update`. It was renamed to
> `jig msync` to make room for the more general passthrough and to avoid
> confusion with [`jig renew`](renew.md), which is jig's own
> template-driven file refresh.
