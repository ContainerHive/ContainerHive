# Test your containers

ContainerHive integrates [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test) by Google to validate built images as part of the build process.

## Writing test definitions

Test definitions are written as Go templates (`test.yml.gotpl`) placed alongside the Dockerfile in the image directory.

```
images/my-image/
├── image.yml
├── Dockerfile
└── test.yml.gotpl
```

The template is rendered with the image context (versions, build args, etc.) during `ch generate`, producing the final test YAML in `dist/`.

### Example

```yaml
schemaVersion: "2.0.0"

fileExistenceTests:
  - name: "Check binary exists"
    path: "/usr/local/bin/my-app"
    shouldExist: true
    permissions: "-rwxr-xr-x"

commandTests:
  - name: "Check version output"
    command: "my-app"
    args: ["--version"]
    expectedOutput: ["{{ .Versions.app }}"]

fileContentTests:
  - name: "Check config file"
    path: "/etc/my-app/config.yml"
    expectedContents: ["production"]
```

Since the test file is a Go template, you can reference versions and build args from your `image.yml`:

```yaml
commandTests:
  - name: "Python version"
    command: "python3"
    args: ["--version"]
    expectedOutput: ["Python {{ .Versions.python }}"]
```

### Variant tests

Variants can have their own test definitions by placing a `test.yml.gotpl` in the variant subdirectory:

```
images/my-image/
├── image.yml
├── Dockerfile
├── test.yml.gotpl
└── slim/
    ├── Dockerfile
    └── test.yml.gotpl
```

## Running tests

After building images, run tests with:

```bash
ch test
```

This will:

1. Collect rendered test definitions from `dist/<image>/<tag>/tests/`.
2. Load or pull the built image for each platform.
3. Execute the tests and produce JUnit XML reports.

### Building before tests

Use `--build` to build images first, then run tests:

```bash
ch test --build
```

You can combine with `--generate` to render templates, build, and test in one step:

```bash
ch test --generate --build
```

### Filtering

You can run tests for specific images or tags:

```bash
ch test my-image:latest
ch test my-image:*
```

When using `--build`, the same filters apply to both the build and test steps.

### CI behavior

In CI environments (`CI` env var is set), ContainerHive automatically starts a local registry and falls back to pulling images from it when no local tar file is found.

## Test reports

JUnit XML reports are written to the platform directory under `dist/`:

```
dist/my-image/latest/linux-amd64/my-image-latest-cst-report.xml
```

These reports can be consumed by CI systems for test result visualization.
