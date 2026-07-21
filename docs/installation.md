# Installation

## Build from source

Requires Go 1.21 or later.

```bash
git clone https://github.com/voxpupuli/jig.git
cd jig
go build -o jig .
```

Move the resulting binary somewhere in your `$PATH`:

```bash
mv jig /usr/local/bin/
```

No other dependencies or runtimes are needed. The exceptions are the
commands that shell out to a Ruby toolchain (`jig validate`,
`jig test unit`, `jig msync`) — for those you need either a working
Ruby/bundler install or a container engine; see
[Running through voxbox](voxbox.md).
