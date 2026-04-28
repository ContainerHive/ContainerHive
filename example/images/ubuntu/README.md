# Ubuntu Image

Base Ubuntu image for container builds.

## Supported Versions

- 22.04 (Jammy Jellyfish)

## Usage

```dockerfile
FROM __hive__/ubuntu:22.04

# Install packages
RUN apt-get update && apt-get install -y curl
```
