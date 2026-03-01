ContainerHive
===
> This project is under active development. It is not yet in any usable state. Use at your own risk.

[![LICENSE](https://img.shields.io/github/license/timo-reymann/ContainerHive)](https://github.com/timo-reymann/ContainerHive/blob/main/LICENSE)
[![CircleCI](https://circleci.com/gh/timo-reymann/ContainerHive.svg?style=shield)](https://app.circleci.com/pipelines/github/timo-reymann/ContainerHive)
[![codecov](https://codecov.io/gh/timo-reymann/ContainerHive/graph/badge.svg?token=5MGUHVhimo)](https://codecov.io/gh/timo-reymann/ContainerHive)
[![GitHub Release](https://img.shields.io/github/v/tag/timo-reymann/ContainerHive?label=version)](https://github.com/timo-reymann/ContainerHive/releases)
[![Renovate](https://img.shields.io/badge/renovate-enabled-green?logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAzNjkgMzY5Ij48Y2lyY2xlIGN4PSIxODkuOSIgY3k9IjE5MC4yIiByPSIxODQuNSIgZmlsbD0iI2ZmZTQyZSIgdHJhbnNmb3JtPSJ0cmFuc2xhdGUoLTUgLTYpIi8+PHBhdGggZmlsbD0iIzhiYjViNSIgZD0iTTI1MSAyNTZsLTM4LTM4YTE3IDE3IDAgMDEwLTI0bDU2LTU2YzItMiAyLTYgMC03bC0yMC0yMWE1IDUgMCAwMC03IDBsLTEzIDEyLTktOCAxMy0xM2ExNyAxNyAwIDAxMjQgMGwyMSAyMWM3IDcgNyAxNyAwIDI0bC01NiA1N2E1IDUgMCAwMDAgN2wzOCAzOHoiLz48cGF0aCBmaWxsPSIjZDk1NjEyIiBkPSJNMzAwIDI4OGwtOCA4Yy00IDQtMTEgNC0xNiAwbC00Ni00NmMtNS01LTUtMTIgMC0xNmw4LThjNC00IDExLTQgMTUgMGw0NyA0N2M0IDQgNCAxMSAwIDE1eiIvPjxwYXRoIGZpbGw9IiMyNGJmYmUiIGQ9Ik04MSAxODVsMTgtMTggMTggMTgtMTggMTh6Ii8+PHBhdGggZmlsbD0iIzI1YzRjMyIgZD0iTTIyMCAxMDBsMjMgMjNjNCA0IDQgMTEgMCAxNkwxNDIgMjQwYy00IDQtMTEgNC0xNSAwbC0yNC0yNGMtNC00LTQtMTEgMC0xNWwxMDEtMTAxYzUtNSAxMi01IDE2IDB6Ii8+PHBhdGggZmlsbD0iIzFkZGVkZCIgZD0iTTk5IDE2N2wxOC0xOCAxOCAxOC0xOCAxOHoiLz48cGF0aCBmaWxsPSIjMDBhZmIzIiBkPSJNMjMwIDExMGwxMyAxM2M0IDQgNCAxMSAwIDE2TDE0MiAyNDBjLTQgNC0xMSA0LTE1IDBsLTEzLTEzYzQgNCAxMSA0IDE1IDBsMTAxLTEwMWM1LTUgNS0xMSAwLTE2eiIvPjxwYXRoIGZpbGw9IiMyNGJmYmUiIGQ9Ik0xMTYgMTQ5bDE4LTE4IDE4IDE4LTE4IDE4eiIvPjxwYXRoIGZpbGw9IiMxZGRlZGQiIGQ9Ik0xMzQgMTMxbDE4LTE4IDE4IDE4LTE4IDE4eiIvPjxwYXRoIGZpbGw9IiMxYmNmY2UiIGQ9Ik0xNTIgMTEzbDE4LTE4IDE4IDE4LTE4IDE4eiIvPjxwYXRoIGZpbGw9IiMyNGJmYmUiIGQ9Ik0xNzAgOTVsMTgtMTggMTggMTgtMTggMTh6Ii8+PHBhdGggZmlsbD0iIzFiY2ZjZSIgZD0iTTYzIDE2N2wxOC0xOCAxOCAxOC0xOCAxOHpNOTggMTMxbDE4LTE4IDE4IDE4LTE4IDE4eiIvPjxwYXRoIGZpbGw9IiMzNGVkZWIiIGQ9Ik0xMzQgOTVsMTgtMTggMTggMTgtMTggMTh6Ii8+PHBhdGggZmlsbD0iIzFiY2ZjZSIgZD0iTTE1MyA3OGwxOC0xOCAxOCAxOC0xOCAxOHoiLz48cGF0aCBmaWxsPSIjMzRlZGViIiBkPSJNODAgMTEzbDE4LTE3IDE4IDE3LTE4IDE4ek0xMzUgNjBsMTgtMTggMTggMTgtMTggMTh6Ii8+PHBhdGggZmlsbD0iIzk4ZWRlYiIgZD0iTTI3IDEzMWwxOC0xOCAxOCAxOC0xOCAxOHoiLz48cGF0aCBmaWxsPSIjYjUzZTAyIiBkPSJNMjg1IDI1OGw3IDdjNCA0IDQgMTEgMCAxNWwtOCA4Yy00IDQtMTEgNC0xNiAwbC02LTdjNCA1IDExIDUgMTUgMGw4LTdjNC01IDQtMTIgMC0xNnoiLz48cGF0aCBmaWxsPSIjOThlZGViIiBkPSJNODEgNzhsMTgtMTggMTggMTgtMTggMTh6Ii8+PHBhdGggZmlsbD0iIzAwYTNhMiIgZD0iTTIzNSAxMTVsOCA4YzQgNCA0IDExIDAgMTZMMTQyIDI0MGMtNCA0LTExIDQtMTUgMGwtOS05YzUgNSAxMiA1IDE2IDBsMTAxLTEwMWM0LTQgNC0xMSAwLTE1eiIvPjxwYXRoIGZpbGw9IiMzOWQ5ZDgiIGQ9Ik0yMjggMTA4bC04LThjLTQtNS0xMS01LTE2IDBMMTAzIDIwMWMtNCA0LTQgMTEgMCAxNWw4IDhjLTQtNC00LTExIDAtMTVsMTAxLTEwMWM1LTQgMTItNCAxNiAweiIvPjxwYXRoIGZpbGw9IiNhMzM5MDQiIGQ9Ik0yOTEgMjY0bDggOGM0IDQgNCAxMSAwIDE2bC04IDdjLTQgNS0xMSA1LTE1IDBsLTktOGM1IDUgMTIgNSAxNiAwbDgtOGM0LTQgNC0xMSAwLTE1eiIvPjxwYXRoIGZpbGw9IiNlYjZlMmQiIGQ9Ik0yNjAgMjMzbC00LTRjLTYtNi0xNy02LTIzIDAtNyA3LTcgMTcgMCAyNGw0IDRjLTQtNS00LTExIDAtMTZsOC04YzQtNCAxMS00IDE1IDB6Ii8+PHBhdGggZmlsbD0iIzEzYWNiZCIgZD0iTTEzNCAyNDhjLTQgMC04LTItMTEtNWwtMjMtMjNhMTYgMTYgMCAwMTAtMjNMMjAxIDk2YTE2IDE2IDAgMDEyMiAwbDI0IDI0YzYgNiA2IDE2IDAgMjJMMTQ2IDI0M2MtMyAzLTcgNS0xMiA1em03OC0xNDdsLTQgMi0xMDEgMTAxYTYgNiAwIDAwMCA5bDIzIDIzYTYgNiAwIDAwOSAwbDEwMS0xMDFhNiA2IDAgMDAwLTlsLTI0LTIzLTQtMnoiLz48cGF0aCBmaWxsPSIjYmY0NDA0IiBkPSJNMjg0IDMwNGMtNCAwLTgtMS0xMS00bC00Ny00N2MtNi02LTYtMTYgMC0yMmw4LThjNi02IDE2LTYgMjIgMGw0NyA0NmM2IDcgNiAxNyAwIDIzbC04IDhjLTMgMy03IDQtMTEgNHptLTM5LTc2Yy0xIDAtMyAwLTQgMmwtOCA3Yy0yIDMtMiA3IDAgOWw0NyA0N2E2IDYgMCAwMDkgMGw3LThjMy0yIDMtNiAwLTlsLTQ2LTQ2Yy0yLTItMy0yLTUtMnoiLz48L3N2Zz4=)](https://renovatebot.com)
[![pre-commit](https://img.shields.io/badge/%E2%9A%93%20%20pre--commit-enabled-success)](https://pre-commit.com/)

<p align="center">
	<img width="512" src="https://raw.githubusercontent.com/timo-reymann/ContainerHive/refs/heads/main/.github/images/logo.png">
    <br />
    Swarm it. Build it. Run it. — Managing container base and library images has never been easier.
</p>

## Features

<!-- List features as bullet points -->

- **Next-gen builds**: Powered by BuildKit, the modern container image builder behind Docker.
- **Multi-platform ready**: Build and push images for any architecture in a single workflow.
- **YAML-driven management**: Define and maintain image sets and variants declaratively.
- **Reproducible layers**: Guarantee consistent, bit-for-bit identical builds every time (given the same inputs)
- **Testing built in**: Validate images as part of the build process, no extra tooling needed.
- **Smart caching**: Optimized caching that works out of the box, no manual tuning required.
- **Enterprise-ready**: Built for scale, compliance, and integration with enterprise workflows (SBOMs, multi-platform,
  structured governance).

## Requirements

- buildkitd
- S3-compatible storage for caching (optional)

## Installation

<!-- Add installation instructions -->

> TBD

## Usage

<!-- Add how to use e.g. code samples etc. -->

> TBD

## Motivation

<!-- Add bit of context why the project has been created -->

I built this project because existing container image management solutions are a mess. Most tools are either wild shell
script collections, ad-hoc manual processes, or overly complex frameworks that don’t integrate well with CI/CD
pipelines. Every time you need to build, test, or manage container images, you end up reinventing the wheel, writing
custom scripts, debugging inconsistent environments, or manually pinning versions. It’s repetitive, error-prone, and
wastes time.

The goal was to create a single, deterministic tool that handles everything from image generation to SBOM creation, with
reproducible builds and minimal host dependencies. No more guessing if your local build matches CI. No more manual
Dockerfile
tweaks for every platform. No more chasing down why a container behaves differently in production. Just a clean,
declarative workflow that works the same everywhere.

This project replaces the chaos with structure: YAML configs for images, automated testing, and
CI pipeline generation. It’s built for engineers who want to focus on shipping features, not fighting toolchains.

## Documentation

<!-- Link for additional documentations -->

Documentation is available at [container-hive.timo-reymann.de](https://container-hive.timo-reymann.de/), hosted on
GitHub Pages.

## Contributing

I love your input! I want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the configuration
- Submitting a fix
- Proposing new features
- Becoming a maintainer

To get started please read the [Contribution Guidelines](./CONTRIBUTING.md).

## Development

### Requirements

<!-- Delete the ones not required -->

- [GNU make](https://www.gnu.org/software/make/)
- [Docker](https://docs.docker.com/get-docker/)
- [pre-commit](https://pre-commit.com/)
- [Go](https://go.dev/doc/install)
- [deterministic-zip](https://github.com/timo-reymann/deterministic-zip)

### Test

```shell
make test-coverage-report
```

### Build

```shell
make build
```

### Credits

<!-- Add work of others etc. -->
Without these libraries this project would not be possible:

- [syft](https://github.com/anchore/syft) by Anchore
- [buildkit](https://github.com/moby/buildkit) by the Moby Project
- [container-struture-test](https://github.com/GoogleContainerTools/container-structure-test) by Google

### Alternatives

<!-- Are there alternatives to use? - List them here -->

- Shell scripts: Hack together docker build, sed, and curl in a file no one dares to refactor.
- Makefiles: Create a 300-line monster with targets like build-% that only work if you set 12 env vars first.
- CI hardcoding: Dump docker commands directly into GitHub Actions/GitLab CI, ensuring local devs can never reproduce
  builds.
- Manual Dockerfiles: Edit tags by hand, ignore SBOMs, and pretend ubuntu:latest is a valid base image.
