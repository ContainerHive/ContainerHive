# Build images with Nix flakes

> [!WARNING]
> The Nix build backend is highly experimental. It is tested with small sample images only. Real projects may hit
> behaviors the samples do not cover. Try it, and file feedback at
> [the issue tracker](https://github.com/ContainerHive/ContainerHive/issues) — any feedback is welcome.

Images in a ContainerHive project can define their filesystem with a Nix flake instead of a Dockerfile. ContainerHive
resolves the project, builds the flake through BuildKit, and pushes, tests, and documents the result like any other
image.

## How it works

ContainerHive runs `nix build` inside a Nix builder image through a custom BuildKit frontend:

1. Discovery finds a `flake.nix` in the image directory and marks the image with the `nix` build backend.
2. `generate` renders the image directory, copying `flake.nix`, `flake.lock`, the `rootfs/` directory, and the test
   config into `dist/`.
3. `build` executes a single executor step inside the configured builder image: `nix build --option sandbox false` in
   `/src`; the flake output is a rootfs directory, which becomes the image filesystem on an empty base.
4. Everything else is unchanged: OCI labels, the S3 or registry build cache, multi-arch pushes, SBOMs, and structure
   tests run like on Dockerfile builds.

Only the BuildKit worker needs to pull the builder image. Neither `ch` nor the CI pipeline needs Nix installed.

## The flake contract

The build produces a **rootfs directory**. The installed attribute is a convention, never user configured:

```text
.#packages.<nix-system>.default
```

The `<nix-system>` maps from the target build platform, so the result does not depend on the system of the builder
image. A flake must define that attribute for every platform it serves; `nixpkgs.lib.systems.flakeExposed` covers all
systems at once:

```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs }:
    {
      packages = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.buildEnv {
            name = "my-tool-rootfs";
            paths = [ pkgs.my-tool ];
          };
        });
    };
}
```

Platform mapping:

| Docker platform | Nix system |
|:----------------|:-----------|
| `linux/amd64`   | `x86_64-linux` |
| `linux/arm64`   | `aarch64-linux` |
| `linux/386`     | `i686-linux` |
| `linux/arm`     | `armv7l-linux` |
| `linux/arm/v6`  | `armv6l-linux` |
| `linux/riscv64` | `riscv64-linux` |

Platforms without a mapping fail the build with a clear error. See
[the range of supported platforms](multi-platform.md) for the project level view.

Keep the parent project files out of the flake. Static files the image should contain have to be part of the flake
output. A `rootfs/` directory next to the flake is not picked up automatically, and `buildEnv` only accepts
derivations in `paths`, so wrap the directory in a derivation and add it to the `buildEnv` above:

```nix
let
  rootfs = pkgs.runCommand "my-tool-rootfs-files" { } "cp -r ${./rootfs} $out";
in
pkgs.buildEnv {
  name = "my-tool-rootfs";
  paths = [ pkgs.my-tool rootfs ];
}
```

## flake.lock handling

The lock file stays optional:

- **With flake.lock**: the build respects it (`--no-update-lock-file`). When the flake input declarations no longer
  match the lock, the build fails with Nix's lock-file error instead of silently re-resolving. A flake.lock also
  stabilizes the BuildKit cache keys, so unchanged images build from cache on every machine that has them.
- **Without flake.lock**: the build resolves inputs at build time (current `HEAD` of pinned github sources, deterministic
  store paths for pins). Discovery logs a warning. Builds are then not reproducible.

Commit a lock file for release workflows: run `nix flake lock` once and commit `flake.lock` next to `flake.nix`.

## Configure the backend in image.yml

Most images need no extra configuration:

```yaml
description: "My tool image, built with Nix"
build: nix

tags:
  - name: 1.0.0
```

Set `build: nix` explicitly, or let discovery pick the backend: a `flake.nix` without a `Dockerfile` uses the Nix
backend automatically. When both files exist, set `build` explicitly to remove the ambiguity.

The `nix` block configures the backend:

```yaml
build: nix

nix:
  builder_image: "nixos/nix:2.31.6"  # image providing the nix binary
  extra_args: [ "--impure" ]       # extra nix build arguments
```

Because a flake has no Dockerfile instructions, the runtime configuration of the image looks like this:

```yaml
config:
  entrypoint: [ "/bin/my-tool" ]
  cmd: [ "--help" ]
  env:
    TOOL_HOME: /var/lib/my-tool
  user: "1000:1000"
  working_dir: /workspace
```

Set `working_dir`, `user`, `cmd`, `env`, and `entrypoint` as needed. Unset fields stay empty.

## Variants

Variants keep working: place a `flake.nix` into the variant directory and configure the variant like an image, with
variant-level `build`, `nix`, and `config` blocks.

## Parent images

A flake build can start from a parent image rootfs instead of an empty base. Set `nix.parent` on the image or variant:

```yaml
build: nix
depends_on:
  - base

nix:
  parent: "__hive__/base:1.0"
```

The value takes two forms:

- `__hive__/<name>:<tag>` — another image of the project. ContainerHive resolves it from the local build cache like a
  Dockerfile dependency; CI resolves it from the registry. Declare the build order with `depends_on`.
- Any other value is a plain image reference, for example `ubuntu:22.04`. It resolves through the registry and is
  digest-pinned like the builder image.

The build mounts the parent rootfs at `/parent` and copies it into the image first, so the flake output overlays the
parent. Two limitations apply:

- The flake cannot read the parent rootfs while it builds. Nix evaluates and builds in its own sandbox; the parent
  merge happens in the build script afterwards. Declare parent-provided file paths as image content, not as flake
  inputs.
- The parent image config (entrypoint, env, user) does not carry over. The `config` block of the image stays
  authoritative.

## Nix store caching

Every build otherwise starts with an empty nix store inside the builder container and re-downloads the closure of a
changed flake. Set `store_cache: true` to keep the store across builds:

```yaml
build: nix

nix:
  store_cache: true
```

The script imports the build closure from a persistent BuildKit cache mount before the build and saves the new
closure after it. A changed flake reuses the downloaded packages instead of re-fetching them. Three limits apply:

- The cache mount lives inside one BuildKit daemon. A fresh daemon starts cold.
- The registry or S3 build cache does not carry the store. For cross-machine warmup, pin the flake inputs with
  `flake.lock` and let nix substitute from a binary cache.
- The saves are best effort and grow unbounded across builds of the daemon. Clear the cache mount when it grows
  too large.

## Known limitations

- Multi-platform `nix build` runs under emulation on non-native machines. Expect slow first builds for arm64 on x86
  builders; BuildKit caches the step afterwards. The build runs with `filter-syscalls = false` because Nix syscall
  filtering does not work under emulation, a known limitation. Understood trade-off when you accept the nix backend.
- The flake cannot read the parent rootfs while it builds (see [parent images](#parent-images)).
- `build_args` do not apply to Nix builds; a flake takes its versions from its own inputs.
