# `jig convert`

Updates `Gemfile`, `Rakefile`, and `spec/spec_helper.rb` in an existing
module to be usable with [voxbox](../voxbox.md) and Vox Pupuli tooling.

```
jig convert
```

Run it from the module's base directory (where `metadata.json` lives —
jig errors out otherwise). It renders the embedded templates for those
three files and overwrites the module's copies, creating `spec/` if it
does not exist:

- `Gemfile`
- `Rakefile`
- `spec/spec_helper.rb`

This is the quickest way to bring a PDK-generated (or hand-maintained)
module onto the toolchain jig's other commands expect, without
re-scaffolding it.

Unlike [`jig renew`](renew.md), `convert` always uses jig's embedded
templates — it does not consult `--template-dir`, `--template-url`, or the
module's `jig.toml`, and it does not require an allowlist. If you want
template-driven, allowlisted updates on an ongoing basis, set up
[`jig renew`](renew.md) instead.
