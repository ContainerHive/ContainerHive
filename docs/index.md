Home
===

<figure markdown="span">
![Logo](./images/logo_with_text_line.png){ width="512" }
</figure>

Swarm it. Build it. Run it. — Managing container base and library images has never been easier.

!!! warning

    This project is under active development. It is not yet in any usable state. Use at your own risk.

## Motivation

Managing container base images and library images at scale is surprisingly painful. Teams end up with scattered Dockerfiles, manual build scripts, inconsistent tagging, and no dependency tracking between images. CI pipelines are hand-rolled per project, caching is an afterthought, and reproducibility is a dream.

ContainerHive grew out of [poc-container-image-manager](https://github.com/timo-reymann/poc-container-image-manager), a Python-based proof of concept that validated the core idea: declarative, YAML-driven image management with dependency resolution, templating, and CI generation. The PoC proved the concept works — but being Python-based, it required a runtime, bundled platform-specific binaries, and wasn't practical to distribute as a single portable tool.

ContainerHive is the production-grade successor, rewritten in Go as a single static binary with no external dependencies beyond BuildKit. It takes the validated ideas from the PoC and packages them into something you can drop into any CI pipeline or developer workstation without setup overhead.

## Features

- **Next-gen builds**: Powered by BuildKit, the modern container image builder behind Docker.
- **Multi-platform ready**: Build and push images for any architecture in a single workflow.
- **YAML-driven management**: Define and maintain image sets and variants declaratively.
- **Reproducible layers**: Guarantee consistent, bit-for-bit identical builds every time (given the same inputs).
- **Testing built in**: Validate images as part of the build process, no extra tooling needed.
- **Smart caching**: Optimized caching via S3 or registry backends, no manual tuning required.
- **SBOM generation**: Generate CycloneDX SBOMs for all built images using Syft.
- **CI pipeline generation**: Generate GitLab CI and GitHub Actions pipelines from your project definition.
- **Bring your own BuildKit**: Connect to any BuildKit instance — local daemon, shared cluster service, or sidecar in a hardened Kubernetes environment.
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

## AI Usage

This project uses AI tooling (Claude Code and Mistral AI models) to assist with development. All AI-generated or AI-assisted changes are **human-reviewed** and applied responsibly — this is not AI slop. Contributors are expected to uphold the same standard: AI tools are welcome, but every change must be understood, reviewed, and owned by the person submitting it.
