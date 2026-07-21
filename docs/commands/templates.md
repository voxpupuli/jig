# `jig templates`

Utility commands for working with templates. See
[Custom templates](../custom-templates.md) for how template overrides and
remote template repositories work.

## `jig templates dump`

Extracts all embedded default templates to a directory on disk. This is
useful as a starting point for creating your own custom templates. If the
destination directory already exists it will be renamed with a timestamp
suffix before writing.

```
jig templates dump <destination>
```

For example:

```bash
jig templates dump ~/.config/jig/templates
```

You can then edit the files in the destination directory and point jig at
them using `--template-dir` or the `template_dir` config key.

## `jig templates resolve`

Shows where a logical template name resolves from, for debugging custom
template setups. It reports which template source is in effect (flags, the
module's `jig.toml`, the config file, or the embedded defaults), every
path checked in order, and the winning file. The lookup is the same one
the scaffolding commands use, so the result is exactly what `jig new`
would render.

```
jig templates resolve <name>
```

For example, inside a module directory:

```console
$ jig templates resolve class/class.pp --template-dir ~/.config/jig/templates
external template directory: /home/me/.config/jig/templates (from --template-dir flag)
  looking for /home/me/.config/jig/templates/class/class.pp.tmpl (external) ... not found
  looking for /home/me/.config/jig/templates/class/class.pp (external) ... not found
not found in external template directory, falling back to embedded templates
  looking for templates/class/class.pp.tmpl (embedded) ... found
resolved class/class.pp to embedded template templates/class/class.pp.tmpl (rendered with text/template)
```

It accepts the same `--template-dir`, `--template-url`, `--template-ref`,
and `--ssh-accept-new` flags as [`jig new`](new.md#template-source-flags),
and exits non-zero if the name does not resolve in any source.
