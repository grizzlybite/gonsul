# Changelog

## v1.0.6 - 2026-07-17

### Added

- Added a dedicated `release_check.yaml` workflow for validating GoReleaser
  configuration and snapshot artifacts on pushes, pull requests, and manual
  runs without creating a GitHub release.
- Added release-focused documentation summarizing the completed modernization
  work, including serialization fixes, deterministic planning, dry-run modes,
  runtime improvements, dependency updates, Docker changes, and verification
  results.
- Added an architecture notes document describing a simpler future
  configuration-to-Consul synchronizer design.

### Changed

- Updated GitHub Actions to current action versions, including
  `actions/checkout@v6`, `actions/setup-go@v6`, and
  `goreleaser/goreleaser-action@v7`.
- Updated release workflow badges in README to point to the current
  `grizzlybite/gonsul` GitHub Actions workflows.
- Configured the release workflow to set up QEMU and Docker Buildx with the
  `docker-container` driver for multi-architecture Docker image publishing.
- Switched Docker image publishing in GoReleaser from the historical
  `minicliposs/gonsul` image name to `grizzlybite/gonsul`.
- Renamed Docker Hub release secrets to `DOCKERHUB_USERNAME` and
  `DOCKERHUB_TOKEN`.
- Configured GoReleaser to use the repository-scoped `secrets.GITHUB_TOKEN`
  for GitHub release publishing.

### Fixed

- Fixed GoReleaser Docker publishing on GitHub Actions by using a Buildx driver
  that supports attestations and multi-platform image builds.
- Fixed release publishing configuration so GitHub releases are created with
  the built-in GitHub token instead of a fine-grained personal token with
  insufficient release permissions.
- Fixed the default dry-run summary output so it is written directly to stdout
  even when the log level suppresses informational logs.
- Fixed Makefile version injection to use the `grizzlybite/gonsul` package
  path for `app.Version` and `app.BuildDate`.
- Fixed README markdown formatting so `make markdownlint` passes with the
  configured line-length limit.

### Tests

- Added unit coverage for writing dry-run summary output directly to an
  arbitrary writer.
- Added CI coverage for GoReleaser `check` and snapshot artifact builds outside
  tag-driven release runs.

### Verification

- `make markdownlint`
- `go test ./...`
- `go vet ./...`
- `goreleaser check`
- `goreleaser build --snapshot --clean`

## v1.0.5 - 2026-07-17

### Added

- Added a GoReleaser v2 based release workflow for tag-driven releases.
- Added `.goreleaser.yaml` with Linux, macOS, and Windows artifacts for
  `amd64` and `arm64`.
- Added a GoReleaser Docker image definition through `Dockerfile.goreleaser`.
- Added `--dry-run-output` with three modes:
  - `summary` for compact dry-run counters;
  - `table` for the previous detailed operation table;
  - `json` for machine-readable dry-run output without KV values.
- Added support for `.yml` files in the YAML validation and expansion path.

### Changed

- `DRYRUN` now defaults to compact `summary` output to avoid flooding stdout on
  large configuration repositories.
- The previous dry-run table output remains available with
  `--dry-run-output=table`.
- README now documents `.yml` support and all dry-run output modes.

### Tests

- Restored and rewrote `internal/config` tests around the current
  `buildConfig` contract.
- Added config coverage for required flags, invalid strategy, invalid
  `allow-deletes`, invalid log level, `--hook-addr`, secrets loading, and
  `--dry-run-output` validation.
- Added exporter coverage proving `.yml` files are expanded like `.yaml` files.
- Added importer coverage for compact dry-run summary and JSON dry-run output.

### Verification

- `goreleaser check`
- `goreleaser build --snapshot --clean`
- `make fmt-check`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- Docker Compose Consul check against the REMD development YAML snapshot:
  1097 expected keys, 1097 actual keys, no missing, extra, changed, or invalid
  array values.

## v1.0.4 - 2026-07-17

### Changed

- Updated the project for the Go 1.26 generation and pinned the toolchain to
  Go 1.26.5 for security scans and release builds.
- Migrated Git operations from the old `gopkg.in/src-d/go-git.v4` module to
  `github.com/go-git/go-git/v5`.
- Refactored application flow so lower-level packages return errors instead of
  terminating the process. `os.Exit` is now kept at the CLI boundary.
- Propagated `context.Context` through the application runtime, importer, and
  Consul HTTP requests.
- Added graceful shutdown support for the hook HTTP server.
- Made the hook HTTP listen address configurable with `--hook-addr`
  (`GONSUL_HOOK_ADDR`), defaulting to `:8000`.
- Refreshed Docker and Compose setup, including `hashicorp/consul:1.22.5` for
  the local Consul service.
- Modernized GitHub Actions for tests, race tests, vet, build, Docker image
  builds, and vulnerability scanning.

### Fixed

- `POLL` mode now keeps running when the only failure is the guarded
  delete-not-allowed condition, matching the documented behavior.
- Consul transaction errors no longer include raw response bodies in logs.
- Git and Consul connection errors now redact sensitive URL credentials, ACL
  tokens, and configured key paths where applicable.
- Updated expanded document internals to use a clearer
  decode -> flatten -> serialize pipeline without changing the v1.0.3 array
  storage contract.

### Tests

- Added integration coverage for Git clone/open/checkout behavior through
  `go-git/v5`.
- Added integration coverage for Consul transaction payloads through a local
  HTTP test server.
- Added tests for hook HTTP error responses and poll-mode delete guard handling.
- Added dependency audit documentation and a `make vulncheck` target that runs
  `govulncheck` under Go 1.26.5.

### Security

- `make vulncheck` reports no reachable vulnerabilities when run with the
  pinned Go 1.26.5 toolchain.
- Updated direct dependencies including `gopkg.in/yaml.v3`, `tablewriter`,
  `mustache`, `testify`, and `gomega`.

### Verification

- `make fmt-check`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `make build`
- `make vulncheck`
- Docker Compose check with `hashicorp/consul:1.22.5`.
- Full Consul load verification against the REMD development YAML snapshot from
  branch `master`: 1097 expected keys, 1097 actual keys, no missing, extra, or
  changed values.

## v1.0.3 - 2026-07-16

### Fixed

- Fixed invalid serialization of arrays from expanded YAML and JSON documents.
  Arrays are now stored as compact JSON array strings, for example:
  `["one","two","other value"]`.
- Preserved array values containing spaces, commas, quotes, backslashes, Unicode
  text, nested arrays, and objects without lossy string formatting.
- Preserved scalar `null` values from expanded YAML and JSON documents as the
  string `null` instead of silently skipping the key.
- Added an explicit validation error for expanded YAML and JSON documents whose
  root is not an object. Root arrays and root scalars now fail with
  `root document must be an object`.
- Avoided logging the configured repository URL when a cloned repository remote
  does not match, reducing the risk of leaking credentials embedded in a URL.

### Changed

- Expanded JSON and YAML arrays now use `encoding/json` serialization instead of
  Go's debug formatting from `fmt.Sprint`.
- Flattening of expanded documents now traverses object keys in sorted order.
- Import operation planning now processes local and live Consul keys in sorted
  order, making dry-run output and transaction ordering deterministic.
- Validation helpers now return errors to callers instead of terminating from
  inside the validation functions.
- README now documents that Consul KV stores expanded arrays as JSON text, not
  native array values.

### Tests

- Added exporter tests for YAML and JSON array serialization.
- Added coverage for empty, numeric, boolean, mixed, nested, object arrays, and
  strings containing spaces, commas, quotes, backslashes, and Unicode.
- Added tests for scalar `null` handling.
- Added tests for root array validation.
- Added importer tests for deterministic operation ordering.
- Checked in stable test mocks so `go test ./...` and `make test` run without
  requiring mock generation during every test run.

### Compatibility Notes

- This release intentionally changes the stored representation of expanded
  arrays from the previous invalid Go slice formatting to valid JSON text.

  Previous behavior:

  ```text
  [one two]
  ```

  New behavior:

  ```json
  ["one","two"]
  ```

- The old format was ambiguous and could not be reliably parsed by standard JSON
  parsers. The new format is considered a bug fix, but consumers that depended
  on the previous raw string representation must be updated.

### Verification

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `make test`
- Integration check with the local Consul service from `docker-compose.yml`.
