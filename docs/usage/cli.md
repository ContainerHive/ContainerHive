# Use the CLI

ContainerHive's behavior can be controlled with various commands and flags.

You can get a full list by running

```bash
ch --help
```

## Global Flags

| Flag | Description |
|:-----|:------------|
| `--project`, `-p` | Project root directory |
| `--build-id` | Build ID appended to tags as `+<id>` |

## Commands

### `generate`

Discover project and render configurations to `dist/`.

```bash
ch generate
```

This must run before `build`. It parses `hive.yml` and all image definitions.

### `build`

Build container images with BuildKit.

```bash
ch build [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--registry` | Enable local registry |
| `--platform` | Override platforms |

You can filter which images to build by passing image:tag patterns as arguments. In CI environments, the registry is auto-enabled.

### `test`

Run container structure tests on built images.

```bash
ch test [image:tag patterns...]
```

Uses [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test) to validate built images against test definitions.

### `sbom`

Generate Software Bill of Materials for built images.

```bash
ch sbom [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--platform` | Override platforms |

Generates CycloneDX JSON format SBOMs using [Syft](https://github.com/anchore/syft) as a Go library. No external tooling required.

### `finalize`

Create multi-arch manifests and semantic version alias tags.

```bash
ch finalize [image:tag patterns...]
```

Creates manifests from platform-specific images and applies semantic version alias tagging. Requires a running registry.

### `verify`

Verify project structure.

```bash
ch verify
```

Validates `hive.yml` and image definitions. Useful as a quick check before building.

### `template ci`

Generate CI pipeline configuration.

```bash
ch template ci --provider <provider> --output <path>
```

| Flag | Description |
|:-----|:------------|
| `--provider` | CI provider (`gitlab` or `github`) |
| `--output` | Output file path |
| `--template-dir` | Custom template directory |
| `--artifacts` | Enable artifact passing |
| `--version` | ContainerHive version override |
| `--image-name` | ContainerHive image name |

### `template custom`

Render custom Go templates with project context.

```bash
ch template custom --template <path.gotpl> --output <path>
```

### `wait`

Wait for infrastructure dependencies to become available.

```bash
ch wait
```

| Flag | Description |
|:-----|:------------|
| `--buildkitd` | Wait for BuildKit daemon |
| `--docker-socket` | Wait for Docker socket |
| `--timeout` | Timeout duration |

### `login`

Log in to a container registry.

```bash
ch login <registry> -u <username> -p <password>
```

| Flag | Description |
|:-----|:------------|
| `--username`, `-u` | Registry username |
| `--password`, `-p` | Registry password |
| `--password-stdin` | Read password from stdin |

### `license`

Show third-party license notices.

```bash
ch license
```
