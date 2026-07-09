# glab-pipelines

Interactive terminal UI for viewing and managing GitLab pipelines through the GitLab CLI (`glab`).

## Prerequisites

- Go 1.22 or newer
- `glab` installed and authenticated
- Access to the GitLab project you want to inspect

Authenticate `glab` before running this tool:

```sh
glab auth login
```

## Build

Build the binary from the repository root:

```sh
go build -o bin/glab-pipelines ./cmd/glab-pipelines
```

Or use the Makefile:

```sh
make build
```

## Run

Run directly with Go:

```sh
go run ./cmd/glab-pipelines
```

Run the built binary:

```sh
./bin/glab-pipelines
```

The repository also includes a convenience launcher that rebuilds into your user cache when source files change:

```sh
./glab-pipelines
```

## Usage

```sh
glab-pipelines                 # active pipelines, default
glab-pipelines running         # only running pipelines
glab-pipelines manual          # only manual pipelines
glab-pipelines pending         # any single status
glab-pipelines all             # newest pipelines regardless of status
glab-pipelines -R group/proj active
```

If you run with `go run`, pass arguments after the package path:

```sh
go run ./cmd/glab-pipelines -- -R group/project active
```

## Configuration

Environment variables:

```sh
GLAB_TUI_LIMIT=10          # newest pipelines to show
GLAB_TUI_REFRESH=20        # pipeline detail refresh interval in seconds
GLAB_TUI_LOG_REFRESH=3     # job log refresh interval in seconds
```

## Key Bindings

Pipeline list:

- `j`/`k` or arrow keys: move selection
- `enter`: open pipeline details
- `r`: refresh
- `q`: quit

Pipeline detail:

- `j`/`k` or arrow keys: move jobs
- `p`: open jobs list
- `s`: start or retry selected job when available
- `c`: cancel selected job when available
- `l`: open logs
- `r`: refresh
- `o`: open pipeline in browser
- `q`: go back

Jobs list:

- `j`/`k` or arrow keys: move selection
- `s`: start or retry selected job when available
- `c`: cancel selected job when available
- `l`: open logs
- `r`: refresh
- `q`: go back

Logs:

- `j`/`k` or arrow keys: scroll
- `pgup`/`pgdn`: page
- `g`: top
- `G`: bottom
- `n`: next job
- `r`: reload
- `q`: go back

## Development

Common commands:

```sh
make fmt
make tidy
make test
make build
```

Equivalent Go commands:

```sh
gofmt -w cmd
go mod tidy
go test ./...
go build -o bin/glab-pipelines ./cmd/glab-pipelines
```

## Repository Layout

```text
cmd/glab-pipelines/   executable application package
go.mod                Go module definition
go.sum                locked module checksums
Makefile              common development commands
glab-pipelines        convenience launcher script
```
