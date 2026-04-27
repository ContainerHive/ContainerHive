# .NET Image

.NET SDK image for building and running .NET applications.

## Supported Versions

- 8.0.100
- 8.0.200
- 8.0.300

## Features

- .NET SDK with multi-channel support
- Build tools included
- Multi-platform support (linux/amd64)

## Usage

```dockerfile
FROM __hive__/dotnet:8.0.100

# Build your application
WORKDIR /app
COPY . .
RUN dotnet build -c Release
```

## Build Args

- `foo` - Example build argument
