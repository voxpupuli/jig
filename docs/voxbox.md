# Running through voxbox

The [`validate`](commands/validate.md), [`test unit`](commands/test.md),
and [`msync`](commands/msync.md) commands run a Ruby toolchain under the
hood. By default that is the host's `bundle`, which needs a working
Ruby/bundler install. Instead, jig can run them inside the
[voxbox](https://github.com/voxpupuli/container-voxbox) container, so the
only host dependency is a container engine. This is especially handy on
Windows, where a system-wide bundler install is awkward.

## Enabling voxbox

Enable it via the `[runner]` section of your
[config file](configuration.md):

```toml
[runner]
type   = "voxbox"                        # "local" (default) or "voxbox"
engine = "docker"                        # container engine: "docker" (default) or "podman"
image  = "ghcr.io/voxpupuli/voxbox:latest"  # container image to run
```

With `type = "voxbox"`, your module root (the current directory) is
mounted at `/repo` inside the container and used as the working
directory, so the toolchain and gems come from the image rather than the
host.

Each setting can also be supplied through the environment, which
overrides the config file:

```bash
export JIG_RUNNER_TYPE=voxbox
export JIG_RUNNER_ENGINE=podman
export JIG_RUNNER_IMAGE=ghcr.io/voxpupuli/voxbox:latest
```

## How the commands map to the container

The voxbox image's entrypoint is already `bundle exec rake`, so the
rake-based commands pass their tasks straight through. `jig test unit`
runs roughly:

```bash
docker run --rm -i -v "$PWD:/repo:Z" -w /repo \
  ghcr.io/voxpupuli/voxbox:latest spec
```

`jig msync` uses `msync`, which is not a rake task, so it overrides the
entrypoint to run bundle directly:

```bash
docker run --rm -i -v "$PWD:/repo:Z" -w /repo --entrypoint bundle \
  ghcr.io/voxpupuli/voxbox:latest exec msync update
```
