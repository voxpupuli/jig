# `jig test`

## `jig test unit`

Runs the module's unit tests. This is a passthrough command that shells
out to `bundle exec rake spec`, so it requires a Ruby toolchain and the
module's bundled gems to be installed — or a container runner (see
[Running through voxbox](../voxbox.md)).

```
jig test unit [args...]
```

Any additional arguments are passed through verbatim to the underlying
rake invocation.
