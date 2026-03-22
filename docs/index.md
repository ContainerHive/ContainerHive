Home
===

<figure markdown="span">
![Logo](./images/logo_with_text_line.png){ width="512" }
</figure>

Swarm it. Build it. Run it. — Managing container base and library images has never been easier.

!!! warning

    This project is under active development. It is not yet in any usable state. Use at your own risk.

## Features

- **Next-gen builds**: Powered by BuildKit, the modern container image builder behind Docker.
- **Multi-platform ready**: Build and push images for any architecture in a single workflow.
- **YAML-driven management**: Define and maintain image sets and variants declaratively.
- **Reproducible layers**: Guarantee consistent, bit-for-bit identical builds every time (given the same inputs).
- **Testing built in**: Validate images as part of the build process, no extra tooling needed.
- **Smart caching**: Optimized caching via S3 or registry backends, no manual tuning required.
- **SBOM generation**: Generate CycloneDX SBOMs for all built images using Syft.
- **CI pipeline generation**: Generate GitLab CI and GitHub Actions pipelines from your project definition.
- **Enterprise-ready**: Built for scale, compliance, and integration with enterprise workflows.

## Supported platforms

The following platforms have prebuilt binaries:

- Linux
    - 64-bit
    - ARM 64-bit
- Darwin
    - 64-bit (Intel)
    - ARM 64-bit (Apple Silicon)
- Docker (x86 & ARM)

!!! info

    New to ContainerHive? Start with the [Get started](./getting-started.md) guide.

## Requirements

- [BuildKit](https://github.com/moby/buildkit) daemon
- S3-compatible storage for caching (optional)
