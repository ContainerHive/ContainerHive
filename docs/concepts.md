# Concepts

The terms used throughout the documentation.

| Term | Meaning |
|:-----|:--------|
| **Project** | A directory with a `hive.yml` at its root and image directories below it. See [Configure your project](configuration/hive.md). |
| **Image** | A directory with an `image.yml` and a `Dockerfile` (or a Nix `flake.nix`). See [Define your images](configuration/image.md). |
| **Tag** | A version of an image, e.g. `1.0.0`. Each tag is built separately and can set its own `versions`. |
| **Variant** | A flavor of an image (e.g. `-selenium`) that layers on top of its tags. |
| **Versions** | Key/value pairs from `image.yml` passed to the build as `<KEY>_VERSION` build arguments. |
| **`__hive__/<image>`** | The `FROM` reference to another image of the project. ContainerHive derives the build order from it. |
| **`dist/`** | Output of `ch generate`: rendered build contexts and configs. `build`, `test` and `sbom` work on it. |
| **Build ID** | Suffix (`-build.<id>`) appended to tags so merge request builds don't clash. See [Build IDs](usage/build-ids.md). |
| **Finalize** | `ch finalize` creates multi-arch manifests and semantic version aliases in the registry. |
| **Shard** | A slice of the images or tests handled by one parallel CI job. See [Sharding](usage/sharding.md). |

## Typical flow

```text
ch generate  →  ch build  →  ch test  →  ch sbom  →  ch finalize
```

See [Build process overview](how-it-works/build-process.md) for what happens in each step.
