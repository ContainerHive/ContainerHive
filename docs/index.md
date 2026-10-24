Home
===

<figure markdown="span">
![Logo](./images/logo_with_text_line.png){ width="512" }
</figure>

Swarm it. Build it. Run it. — Managing container base and library images has never been easier.

ContainerHive turns a directory of Dockerfiles into a dependency-aware build: it works out the build order, builds with BuildKit, runs your tests, generates SBOMs and writes the CI pipeline.

```
images/
├── base/      # Dockerfile: FROM ubuntu:24.04
└── node/      # Dockerfile: FROM __hive__/base
```

```bash
ch generate && ch build && ch test
```

<div class="grid cards" markdown>

- **New here?** Follow [Get started](./getting-started.md) and have a first build running in minutes. Wondering if this is for you? Read [Why use ContainerHive?](./why-containerhive.md)
- **Already using it?** Jump to the [CLI reference](./usage/cli.md), [`hive.yml`](./configuration/hive.md), [`image.yml`](./configuration/image.md) or [CI integration](./usage/ci-integration.md).

</div>

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

## Requirements

- [BuildKit](https://github.com/moby/buildkit) daemon (`ch dev buildkitd start` can run one for you)
- S3-compatible storage for caching (optional)

Prebuilt binaries are available for Linux and macOS (64-bit and ARM 64-bit), plus a Docker image. See [Install ContainerHive](./installation.md).
