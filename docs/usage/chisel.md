# Build images with Chisel

> [!WARNING]
> The Chisel build backend is highly experimental, like the Nix build backend. It is tested with small sample images
> only. Real projects may hit behaviors the samples do not cover. Try it, and file feedback at
> [the issue tracker](https://github.com/ContainerHive/ContainerHive/issues) — any feedback is welcome.

Images in a ContainerHive project can define their filesystem with [Chisel](https://github.com/canonical/chisel)
instead of a Dockerfile. Chisel cuts minimal package slices from the Ubuntu archives (see
[canonical/chisel-releases](https://github.com/canonical/chisel-releases)) and assembles them into a rootfs.
ContainerHive resolves the project, cuts the rootfs through BuildKit, and pushes, tests, and documents the result
like any other image.

## How it works

ContainerHive runs `chisel cut` inside a builder image through a custom BuildKit frontend:

1. Discovery finds a `chisel.yaml` cut spec in the image directory and marks the image with the `chisel` build
   backend.
2. `generate` renders the image directory, copying the cut spec, the custom `slices/` directory, the `rootfs/`
   directory, and the test config into `dist/`.
3. `build` prepares a cached toolchain stage on top of the builder image: git, ca-certificates, and the chisel binary,
   each only when the image lacks it. A second stage then runs `chisel cut --release <local-checkout>` in `/src`; the cut rootfs
   becomes the image filesystem on an empty base.
4. Everything else is unchanged: OCI labels, the S3 or registry build cache, multi-arch pushes, SBOMs, and
   structure tests run like on Dockerfile builds.

Only the BuildKit worker pulls the builder image and the chisel tool binary. Neither `ch` nor the CI pipeline needs
Chisel or git installed. ContainerHive does not bundle a builder image; the default one comes from the helper-images
project.

## The cut spec contract

The build entrypoint is `chisel.yaml` next to `image.yml`:

```yaml
release: ubuntu-24.04
slices:
  - base-files_base
  - base-files_release-info
  - ca-certificates_data
  - hello_bins
```

- **release** names a branch of the canonical `chisel-releases` repository, for example `ubuntu-24.04`. Chisel cuts
  release `ubuntu-24.04` for the `ubuntu-24.04` branch in the archive.
- **slices** lists the slices to cut in chisel notation `<package>_<slice>`. Chisel resolves slice dependencies (the
  `essential` edges) automatically.

A JSON schema for the cut spec is published to
[SchemaNest](https://schema-nest.timo-reymann.de/schemas/json-schema/containerhive-chisel/latest?tab=setup) and can be
referenced directly at
[schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-chisel/latest](https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-chisel/latest).

An optional `ignore` field lets chisel cut unstable or unmaintained releases, same behavior as `chisel cut
--ignore`. See the [cut command reference](https://github.com/canonical/chisel).

> [!NOTE]
> A common naming collision: Chisel itself also has a `chisel.yaml` — the release manifest inside a
> chisel-releases checkout. The two formats are unrelated. ContainerHive's `chisel.yaml` is a cut spec that drives
> the build; the release manifest lives inside the release checkout that `chisel cut` consumes.

## Release sources

The release supports two forms:

- **A branch name** — `release: ubuntu-24.04` clones the named branch of `canonical/chisel-releases`. The build
  tracks the branch head, like Chisel's own release fetch.
- **A git mapping** — for pinned and custom releases:

  ```yaml
  release:
    git: https://github.com/canonical/chisel-releases
    ref: "0a817bd"  # commit, tag, or branch
  ```

  A missing `ref` checks out the default branch. Discovery logs a warning, because the build is then not
  reproducible. Pin a commit for release workflows.

> [!TIP]
> Rebuilds fetch the release checkout incrementally from a cached clone, so an unchanged release skips the checkout
> and the deb downloads. The git path also works offline once the clone and the deb cache are warm, and it avoids
> the HTTP check that Chisel's built-in release fetch makes on every run.

## Custom slice definitions

An optional `slices/` folder next to the cut spec holds custom slice definitions in the
[chisel-releases format](https://github.com/canonical/chisel-releases) — one YAML per package, with `package:`,
`slices:`, and the `contents` paths. The build overlays the folder on the resolved release before the cut, so the
new packages resolve like release definitions:

```text
images/hello/
├── chisel.yaml
└── slices/
    └── mypkg.yaml
```

Cut a locally defined package like any other: add `mypkg_bins` to `slices` in the cut spec.

## Configure the backend in image.yml

Most images need no extra configuration:

```yaml
description: "My tool image, cut from Ubuntu slices"
build: chisel

tags:
  - name: 1.0.0
```

Set `build: chisel` explicitly, or let discovery pick the backend: a `chisel.yaml` that is the only build
entrypoint uses the Chisel backend automatically. A cut spec next to a `flake.nix` or a `Dockerfile` is ambiguous
and fails discovery; set `build` in that case.

The `chisel` block configures the backend:

```yaml
build: chisel

chisel:
  builder_image: "ghcr.io/containerhive/helper-images/chisel:1.5"  # base image for the build stages (default)
  chisel_version: "v1.5.1"       # chisel tool version, only downloaded when the builder image ships another one
  extra_args: []                 # extra chisel cut arguments
```

The default builder image is the `chisel` image of
[ContainerHive/helper-images](https://github.com/ContainerHive/helper-images). It ships chisel, git and
ca-certificates, so the toolchain stage installs and downloads nothing. That repository is itself a ContainerHive
project and publishes one tag per chisel release series (`1.5`, `1.4`, ...).

The toolchain stage adds only what the builder image lacks:

- **git** and **ca-certificates** are installed with `apt-get` when missing, so such an image has to be Debian or
  Ubuntu based. An image that already contains both needs no package manager.
- **chisel** is downloaded from the GitHub release when the image ships no chisel, or ships another version than
  `chisel_version`. An image declares the chisel it ships with the `chisel.version` label. Without
  `chisel_version`, a shipped chisel is used as is; without a label, the default version is downloaded.

A plain `ubuntu:24.04` still works as builder image.

Because a cut spec carries no Dockerfile instructions, the runtime configuration of the image looks like this:

```yaml
config:
  entrypoint: [ "/bin/hello" ]
  cmd: [ "--help" ] 
  env:
    TOOL_HOME: /var/lib/my-tool
  user: "1000:1000"
  working_dir: /workspace
```

Set `working_dir`, `user`, `cmd`, `env`, and `entrypoint` as needed. Unset fields stay empty.

## Variants

Variants keep working: place a `chisel.yaml` into the variant directory and configure the variant like an image,
with variant-level `build`, `chisel`, and `config` blocks.

## Parent images

A cut build can start from a parent image rootfs instead of an empty base. Set `chisel.parent` on the image or
variant:

```yaml
build: chisel
depends_on:
  - base

chisel:
  parent: "__hive__/base:1.0"
```

The value takes three forms:

- `__hive__/<name>:<tag>` — another image of the project. ContainerHive resolves it from the local build cache like
  a Dockerfile dependency; CI resolves it from the registry. The build order derives from the reference; an
  explicit `depends_on` is not required.
- `__hive_parent__` — the variant placeholder. Only valid inside a variant: it resolves to the base tag
  `__hive__/<name>:<tag>` of the same image.
- Any other value is a plain image reference, for example `ubuntu:22.04`. It resolves through the registry and is
  digest-pinned like the builder image.

The build mounts the parent rootfs at `/parent` and copies it into the image first, so the cut output overlays the
parent. The slice files land on top of the parent files. An image-config-level merge applies as well: entrypoint,
cmd, env, user and working dir inherit from the parent unless the `config` block sets them. Symlinks stay symlinks:
the cut output is copied verbatim, and only the parent merge resolves links into the target directory.

## Caching

Chisel caches two things inside one BuildKit worker container: the git checkout of the release repo, and the
downloaded `.deb` packages. Set `cache: false` to opt out when a daemon has no place for a growing cache:

```yaml
build: chisel

chisel:
  cache: false
```

The default is on. The cache lives inside one BuildKit daemon. A fresh daemon starts cold, and the registry or S3
build cache does not carry it — the images themselves are what the build cache stores. The cache grows with the
deb churn of the archive; prune the `hive-chisel-cache` mount when it grows too large.

## Static rootfs files

A `rootfs/` directory next to the cut spec contributes static image content — files chisel cannot know, like
configuration files. The build assembles the filesystem in this order:

1. The parent rootfs (when `chisel.parent` is set).
2. The cut slice files.
3. The `rootfs/` files.

`rootfs/` files overwrite slice files on the same path. `structure test` configs and OCI labels behave the same on
all backends.

## Templating

A cut spec renders through the ContainerHive template engine like any other build entrypoint. Name the templated
spec `chisel.yaml.gotpl`; `generate` renders it and writes the result as `chisel.yaml`. A plain `chisel.yaml` is
copied as is without template processing.

```yaml title="chisel.yaml.gotpl"
release: ubuntu-{{ .Versions.ubuntu }}
slices:
  - base-files_base
  - hello_bins
```

Rendering happens at `generate` time. A new build arg value re-renders the spec and invalidates the build cache, by
design.

## Known limitations

- Ubuntu Pro archives (`esm-apps`, `esm-infra`, `fips`) are not supported until someone requests them. FIPS builds
  need credentials this backend does not wire up yet.
- Release format support follows the chisel tool version: the pinned `chisel_version` opens the formats, see
  [the compatibility matrix](https://github.com/canonical/chisel#chisel-releases).
- Multi-platform builds run under emulation on non-native machines, like the Nix backend. Chisel publishes no
  linux/386 tool binary, so `linux/386` images fail the build with a clear error.
- The cut cannot read the parent rootfs while it runs. The parent merge happens in the build script afterwards.
- The release checkout and the deb cache live in one cache mount. Chisel cuts on one daemon are serialized, so
  concurrent builds never race on the release checkout.
