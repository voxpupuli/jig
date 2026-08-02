# `jig version`

Print the current jig version and exit. This command does not load or
validate any configuration file, so it works even when `jig.toml` is
broken or missing.

```
jig version
```

The output depends on how the binary was built:

- **Release build** — stamped semantic version (e.g. `jig 2.1.0`).
- **Built from a git checkout** — `jig dev (commit abc123def456)`, or `jig dev (commit abc123def456+dirty)` when the working tree has uncommitted changes.
- **`go install` at a specific version** — the module version from the build info.
- **Plain `go build` / `go install` without version info** — `jig dev`.
