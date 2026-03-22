# Define your tests

Test definitions validate built images using [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test). They are rendered during `ch generate` and executed by `ch test`.

## File location

Place a `test.yml` or `test.yml.gotpl` file alongside the Dockerfile in the image directory. The `.gotpl` extension is optional — use it when you need Go template variables (versions, build args). Plain `test.yml` files are copied as-is.

```
images/my-image/
├── image.yml
├── Dockerfile
└── test.yml.gotpl
```

Variants can have their own test definitions:

```
images/my-image/
├── image.yml
├── Dockerfile
├── test.yml.gotpl
└── slim/
    ├── Dockerfile
    └── test.yml.gotpl
```

When both an image-level and variant-level test file exist, both are collected and executed.

## Template context

The template receives a `TemplateContext` with the following fields:

| Field | Type | Description |
|:------|:-----|:------------|
| `Versions` | map | Version values resolved from image, tag, and variant definitions |
| `BuildArgs` | map | Build arguments resolved from image, tag, and variant definitions |
| `ImageName` | string | The name of the image |

Values follow a merge hierarchy: **image defaults -> tag overrides -> variant overrides**.

## Schema

Test files use the container-structure-test schema version `2.0.0`. The following test types are supported:

### Command tests

Run a command inside the container and check the output:

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "Python version"
    command: "python"
    args: ["--version"]
    expectedOutput: ["Python {{ .Versions.python }}"]
```

### File existence tests

Check that files exist (or don't) with expected permissions:

```yaml
schemaVersion: 2.0.0
fileExistenceTests:
  - name: "Binary exists"
    path: "/usr/local/bin/my-app"
    shouldExist: true
    permissions: "-rwxr-xr-x"
```

### File content tests

Check the contents of files in the image:

```yaml
schemaVersion: 2.0.0
fileContentTests:
  - name: "Config contains production"
    path: "/etc/my-app/config.yml"
    expectedContents: ["production"]
```

### Metadata tests

Validate image metadata like labels and environment variables:

```yaml
schemaVersion: 2.0.0
metadataTest:
  labels:
    - key: "app"
      value: "{{ .ImageName }}"
  envVars:
    - key: "APP_VERSION"
      value: "{{ .Versions.app }}"
```

## Template functions

Templates have access to all [Sprig](http://masterminds.github.io/sprig/) functions plus:

| Function | Description |
|:---------|:------------|
| `resolve_base(name, tag)` | Produces an internal image reference for inter-image dependencies |

## Examples

### Version check with Go template variable

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "go-version"
    command: "go"
    args: ["version"]
    expectedOutput: ["go{{ .Versions.go }}"]
metadataTest:
  labels:
    - key: "app"
      value: "{{ .ImageName }}"
```

### Rootfs file validation

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "python-version"
    command: "python"
    args: ["--version"]
    expectedOutput: ["Python {{ .Versions.python }}"]

fileExistenceTests:
  - name: "rootfs-python-info-exists"
    path: "/etc/python-info"
    shouldExist: true

fileContentTests:
  - name: "rootfs-python-info-has-version-level-content"
    path: "/etc/python-info"
    expectedContents: ["level=version"]
```

### Variant-specific test

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "node-version"
    command: "node"
    args: ["--version"]
    expectedOutput: ["{{ .Versions.nodejs }}"]
```

## Rendered output

During `ch generate`, test templates are rendered to `dist/<image>/<tag>/tests/`:

```
dist/my-image/latest/tests/
├── image.yml       # from image-level test.yml.gotpl
└── variant.yml     # from variant-level test.yml.gotpl (if present)
```

These rendered files are consumed by `ch test` using container-structure-test.

For more on running tests, see [Container Structure Tests](../usage/testing.md).
