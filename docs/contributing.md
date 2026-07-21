# Contributing

Contributions are welcome. The project is in early stages, so the best
place to start is by opening an issue to discuss what you want to work on
before sending a PR.

## Project layout

```
.
├── main.go
├── commands/        # Cobra command definitions
└── internal/
    ├── build/
    ├── bundle/      # bundle/rake runner (local or voxbox container)
    ├── config/
    ├── forge/
    ├── module/      # Module metadata and validation
    ├── release/
    ├── remote/      # Remote template repository fetching
    ├── scaffold/    # Scaffolding orchestration
    └── template/    # Template rendering with fallback logic
        └── templates/  # Embedded default templates
```

## Testing

Run the full test suite with:

```bash
go test ./...
```

Tests live alongside the source files they cover (`*_test.go`), which is
the standard Go convention.

A few patterns used throughout the test suite that contributors should
follow:

- **Table-driven tests** for functions with multiple input variations.
  Use a `cases := []struct{...}` slice and `t.Run` for each case.
- **`t.TempDir()`** for any test that touches the filesystem. It is
  cleaned up automatically after the test and requires no
  `defer os.Remove`.
- **`fakeRenderer`** in `internal/scaffold` implements the
  `scaffold.Renderer` interface and can be used to test template
  rendering paths without hitting the real embedded templates.
- **`makeBuildDir`** in `internal/build`, **`makeModuleDir`** in
  `internal/scaffold`, and **`makeModuleDir`** in `internal/release` are
  shared helpers that create realistic on-disk module structures for
  tests that need them. **`fakePublisher`** in `internal/release`
  implements the `forge.Publisher` interface for testing the release
  sequence without making real HTTP calls.
- Both characterization tests (pinning current behavior) and adversarial
  tests (checking rejection of invalid or malicious input) are expected.
  When adding a new feature, include both.

## Git hooks

A pre-commit hook is provided in `githooks/` that runs `gofmt`, `go vet`,
`go test ./...`, and `govulncheck ./...` before each commit. Enable it
with:

```bash
git config core.hooksPath githooks
```

## Design notes

- Templates are embedded via `go:embed`. External templates take
  precedence over embedded ones, with per-file fallback to embedded
  defaults when a custom template is not found. Template names are
  validated to prevent path traversal before any file is read.
- `--force` never deletes existing files outright. It creates a
  timestamped backup of the target directory first.
- Module name validation uses a `ValidationResult` type with an
  iota-based `Severity`. Violations at the `Warning` level do not halt
  execution. Version strings must be valid semver (`MAJOR.MINOR.PATCH`).
  URL fields (`source`, `project_page`, `issues_url`) must use `http` or
  `https` schemes when present; invalid URLs are errors that abort the
  build and release.
- The Forge HTTP client (`internal/forge`) is hidden behind a
  `Publisher` interface so the release sequence can be tested without
  making real network requests.
- Component names (module names, class names, defined type names) are
  validated to reject empty strings, path separators, and traversal
  sequences before they are used to construct filesystem paths.
- `os.Getwd()` is called only in the `commands/` layer. Internal
  packages receive directory paths as arguments, which keeps them
  testable without manipulating the process working directory.
- Trust decisions (like `ssh_accept_new`) are read only from the user's
  config, environment, or flags — never from files shared through a
  module repository (`jig.toml`, `metadata.json`).
- Config is handled with [Viper](https://github.com/spf13/viper).
