# Set up local development

If you want to work on ContainerHive you can do so, easily.

Before starting to develop make sure you have read
the [Contribution Guidelines](https://github.com/ContainerHive/ContainerHive/blob/main/CONTRIBUTING.md).

## Requirements

Make sure to install the following tools:

- [Go](https://go.dev/doc/install)
- [GNU make](https://www.gnu.org/software/make/)
- [Docker](https://docs.docker.com/get-docker/)
- [pre-commit](https://pre-commit.com/)

## Run

```sh
go run ./cmd/ch/ -h
```

## Test

Run tests and show the coverage report in your default browser.

```sh
make test-coverage-report
```

Run specific package tests:

```sh
go test ./internal/dependency/...
```

Skip integration tests:

```sh
go test -short ./...
```

## Build

Build binaries for all platforms.

```sh
make build
```

## Generate embedded resources

Required before building.

```sh
make generate
```

## Preview documentation

Preview the properdocs documentation locally with hot-reload for changes.

```sh
pip install -r requirements-docs.txt
properdocs serve
```
