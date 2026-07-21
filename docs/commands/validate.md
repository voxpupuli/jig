# `jig validate`

Runs validation checks against the current module. By default this mirrors
`pdk validate`: syntax (`rake validate`), puppet-lint (`rake lint`) and
rubocop (`rake rubocop`) all run, each as its own rake invocation.

```
jig validate [flags] [-- args...]
```

It shells out to `bundle exec rake <task>` per check, so it requires a
Ruby toolchain and the module's bundled gems to be installed — or a
container runner (see [Running through voxbox](../voxbox.md)).

**Flags:**

| Flag | Description |
| ---- | ----------- |
| `-s`, `--syntax` | Run syntax checks (`rake validate`) |
| `-l`, `--lint` | Run puppet-lint checks (`rake lint`) |
| `-r`, `--rubocop` | Run rubocop checks (`rake rubocop`) |

With no flags all three checks run. Passing one or more flags runs only
the selected checks, e.g. `jig validate -s` for syntax only or
`jig validate -sl` to skip rubocop. Checks run in order and stop at the
first failure, propagating its exit code.

Any arguments after `--` are passed through verbatim to each underlying
rake invocation.
