# `jig convert`

Brings an existing module — PDK-generated, hand-maintained, or predating
`metadata.json` entirely — onto the toolchain jig's other commands and
[voxbox](../voxbox.md) expect.

```
jig convert
```

Run it from the module's base directory. What it does with `metadata.json`
depends on what's there:

- **Missing.** jig creates it. If a `Modulefile` (the Puppet 3.x-era module
  metadata format) is present, its `name`, `version`, `author`, `license`,
  `summary`/`description`, `source`, and `dependency` lines pre-fill the new
  `metadata.json`, and the `Modulefile` is left in place with a warning that
  it's no longer used. Otherwise, jig runs the same interview `jig new
  module` runs (honoring `--skip-interview` and the `-u/-a/-l/-s/-S` flags
  below), and the module name is derived from the current directory's name
  using the Forge convention (`puppet-nftables` and `binford2k-demo` both
  give `nftables`/`demo`; a directory with no `-` is used as-is). A flag
  always wins over the `Modulefile`, which always wins over your
  [config file](../configuration.md) defaults. `jig.toml` is written too, if
  it's also missing.
- **Present but not valid JSON.** jig reports the parse error and stops. It
  won't guess at a broken file, and it leaves `Gemfile`, `Rakefile`, and
  `spec/spec_helper.rb` untouched.
- **Present and parses, but incomplete.** jig fills in the keys it has a
  safe default for — `version` (`0.1.0`), and empty `dependencies`,
  `requirements`, `operatingsystem_support`, and `tags` lists — and warns
  about anything else validation flags (like a missing `author` or
  `license`), without a safe default to fall back on. It never overwrites a
  value that's already present, so running `convert` twice is a no-op.
- **Present and valid.** Unchanged: jig doesn't touch it.

In every case except a broken `metadata.json`, jig then renders the embedded
templates for these three files and overwrites the module's copies, creating
`spec/` if it does not exist:

- `Gemfile`
- `Rakefile`
- `spec/spec_helper.rb`

**Flags:**

| Flag | Description |
|------|-------------|
| `-u, --forge-user` | Forge username (used only if `metadata.json` must be created) |
| `-a, --author` | Author name (used only if `metadata.json` must be created) |
| `-l, --license` | License type (used only if `metadata.json` must be created) |
| `-s, --summary` | Summary of the module (used only if `metadata.json` must be created) |
| `-S, --source` | Source URL for the module (used only if `metadata.json` must be created) |
| `-i, --skip-interview` | Skip the interview when `metadata.json` must be created |
| `--dry-run` | Show what would change without writing any files |

Unlike [`jig renew`](renew.md), `convert` always uses jig's embedded
templates — it does not consult `--template-dir`, `--template-url`, or the
module's `jig.toml`, and it does not require an allowlist. If you want
template-driven, allowlisted updates on an ongoing basis, set up
[`jig renew`](renew.md) instead.
