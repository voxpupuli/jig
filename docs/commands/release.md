# `jig release`

Validates metadata, sets the version, builds the module package, and
publishes it to the Puppet Forge.

```
jig release [flags]
```

The release sequence is:

1. Validate the version string and module metadata (unless
   `--skip-validation`).
2. Write the new version into `metadata.json`.
3. Build the module package (unless `--skip-build`).
4. Upload the package to the Forge (unless `--skip-publish`).

A Forge API token is required for publishing. Set `forge_token` in your
[config file](../configuration.md) or pass it via `--token`. You can
generate a token from your account page on the
[Puppet Forge](https://forge.puppet.com).

**Flags:**

| Flag | Description |
|------|-------------|
| `-v, --version` | Version to release, e.g. `1.2.3` (required) |
| `-k, --token` | Forge API token (overrides `forge_token` in config) |
| `--skip-validation` | Skip metadata validation |
| `--skip-build` | Skip building the module archive |
| `--skip-publish` | Skip publishing to the Forge |

If `--skip-build` is set without `--skip-publish`, jig expects the archive
to already exist under `pkg/`. An error is returned if it is not found.

Version strings must be valid semver (`MAJOR.MINOR.PATCH`).
