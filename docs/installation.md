# Installation

jig ships as a single static binary. Pick whichever of the methods below
suits your platform:

- [Download a release](#download-a-release) — prebuilt packages and
  archives for Linux, macOS, and Windows
- [Install with `go install`](#install-with-go-install) — if you already
  have a Go toolchain
- [Build from source](#build-from-source) — for development or unreleased
  changes

No other dependencies or runtimes are needed. The exceptions are the
commands that shell out to a Ruby toolchain (`jig validate`,
`jig test unit`, `jig msync`) — for those you need either a working
Ruby/bundler install or a container engine; see
[Running through voxbox](voxbox.md).

## Download a release

Every [GitHub release](https://github.com/voxpupuli/jig/releases) carries
the following assets, where `<version>` is the release number without the
leading `v` (e.g. `2.4.0`):

| Platform | Architectures | Files |
|---|---|---|
| Debian, Ubuntu, and derivatives | `amd64`, `arm64` | `jig_<version>_linux_<arch>.deb` |
| RHEL, Fedora, SUSE, and derivatives | `amd64`, `arm64` | `jig_<version>_linux_<arch>.rpm` |
| Linux (any distribution) | `amd64`, `arm64` | `jig_<version>_linux_<arch>.tar.gz` |
| macOS | `amd64` (Intel), `arm64` (Apple silicon) | `jig_<version>_darwin_<arch>.tar.gz` |
| Windows | `amd64` | `jig_<version>_windows_amd64.zip` |

Each release also includes `sha256sums.txt` with checksums for all of the
above.

### Verify the download

```bash
sha256sum --ignore-missing -c sha256sums.txt
```

On macOS, use `shasum -a 256 --ignore-missing -c sha256sums.txt`.

### Debian / Ubuntu (`.deb`)

```bash
VERSION=2.4.0
curl -LO "https://github.com/voxpupuli/jig/releases/download/v${VERSION}/jig_${VERSION}_linux_amd64.deb"
sudo apt install "./jig_${VERSION}_linux_amd64.deb"
```

### RHEL / Fedora / SUSE (`.rpm`)

```bash
VERSION=2.4.0
sudo dnf install "https://github.com/voxpupuli/jig/releases/download/v${VERSION}/jig_${VERSION}_linux_amd64.rpm"
```

On SUSE, use `sudo zypper install` with the same URL; on older systems
without `dnf`, download the file and run `sudo rpm -i` on it.

Both packages install the binary to `/usr/bin/jig`.

> **Upgrades are manual.** The packages are published as release assets,
> not through an apt or yum/dnf repository, so `apt upgrade` and
> `dnf upgrade` will not pick up new versions. To upgrade, download the
> newer package and install it the same way — the package manager replaces
> the old version in place.

### Linux and macOS archives (`.tar.gz`)

```bash
VERSION=2.4.0
OS=linux     # or darwin
ARCH=amd64   # or arm64
curl -LO "https://github.com/voxpupuli/jig/releases/download/v${VERSION}/jig_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "jig_${VERSION}_${OS}_${ARCH}.tar.gz" jig
sudo mv jig /usr/local/bin/
```

#### macOS Gatekeeper

The macOS binaries are not signed or notarized, so a copy downloaded
through a browser is quarantined and Gatekeeper refuses to run it
("jig cannot be opened because the developer cannot be verified").
Clear the quarantine attribute after extracting:

```bash
xattr -d com.apple.quarantine /usr/local/bin/jig
```

Alternatively, run it once, then allow it under **System Settings →
Privacy & Security**. Files downloaded with `curl` are not quarantined, so
the command-line steps above avoid the prompt entirely.

### Windows (`.zip`)

Download `jig_<version>_windows_amd64.zip` from the release page, extract
`jig.exe`, and place it in a directory on your `PATH`.

## Install with `go install`

With a Go toolchain (Go 1.25 or later) on your machine:

```bash
go install github.com/voxpupuli/jig/v2@latest
```

Replace `@latest` with a tag such as `@v2.4.0` to pin a version. The binary
lands in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` when `GOBIN` is unset;
make sure that directory is on your `PATH`.

## Build from source

Requires Go 1.25 or later.

```bash
git clone https://github.com/voxpupuli/jig.git
cd jig
go build -o jig .
```

Move the resulting binary somewhere in your `$PATH`:

```bash
mv jig /usr/local/bin/
```

## Verify the installation

```bash
jig version
```
