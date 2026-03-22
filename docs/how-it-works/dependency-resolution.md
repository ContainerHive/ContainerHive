How dependency resolution works
==================

ContainerHive builds images in the correct order by resolving dependencies between them.

## How dependencies are discovered

Dependencies are resolved from two sources:

1. **Explicit declaration**: The `depends_on` field in `image.yml` lists images this image depends on.
2. **Automatic detection**: ContainerHive scans `FROM` statements in Dockerfiles to detect references to other images in the project.

Both sources are combined to build a directed acyclic graph (DAG) that determines the build order.

## Build order

Images are grouped into layers based on their dependency depth:

1. Images with no dependencies are built first (depth 0).
2. Images depending only on depth-0 images are built next (depth 1).
3. This continues until all images are built.

Images at the same depth can be built in parallel.

## Inter-image dependencies

When an image depends on another image in the project, ContainerHive uses OCI layout named contexts to pass the built image as a build context. This avoids needing to push intermediate images to a remote registry.

## CI pipeline generation

The dependency graph is also used to generate CI pipelines. Each depth level becomes a stage in the generated pipeline, allowing the CI system to parallelize builds within each stage.
