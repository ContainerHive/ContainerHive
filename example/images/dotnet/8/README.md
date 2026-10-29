# .NET Image

.NET SDK image for building and running .NET applications.

## Supported Versions

- 8.0
- 9.0
- 10.0

## Features

- .NET SDK, tags generated from the official release feeds (`tag_ranges`)
- Build tools included
- Multi-platform support (linux/amd64)

## Usage

```dockerfile
FROM __hive__/dotnet:8.0

# Build your application
WORKDIR /app
COPY . .
RUN dotnet build -c Release
```

## Build Args

- `foo` - Example build argument
