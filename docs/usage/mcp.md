# MCP Server

ContainerHive provides an MCP (Model Context Protocol) server that enables AI assistants to manage container images
programmatically.

## Usage

Start the MCP server:

```bash
ch mcp --project /path/to/project
```

The server uses stdio transport, making it compatible with MCP clients like Claude Code, Cursor, and other AI
assistants.

## Tools

### list_images

List all images in the ContainerHive project.

**Parameters:** None

**Returns:** Array of images with name, description, tags, variants, versions, and platforms.

### get_image

Get full configuration details for a specific image.

**Parameters:**

- `name` (string, required): Name of the image

**Returns:** Image configuration including tags, variants, versions, build args, depends_on, and platforms. Secrets are
excluded.

### get_dependencies

Get build dependencies for an image.

**Parameters:**

- `name` (string, required): Name of the image
- `direction` (string, required): "forward" for dependencies, "reverse" for dependents

**Returns:** Ordered array of image names.

### get_image_schema

Get the JSON schema for image.yml configuration files.

**Parameters:** None

**Returns:** JSON schema string.

### get_hive_schema

Get the JSON schema for hive.yml configuration files.

**Parameters:** None

**Returns:** JSON schema string.

### add_image

Create a new image directory with a stub Dockerfile and image.yml.

**Parameters:**

- `name` (string, required): Name of the image to create
- `description` (string, required): Description of the image
- `base_tag` (string, required): Base Docker tag (e.g., ubuntu:22.04)
- `dockerfile_content` (string, optional): Custom Dockerfile content

**Returns:** Confirmation message.

### add_image_variant

Add a new variant to an existing image with a stub Dockerfile.

**Parameters:**

- `image_name` (string, required): Name of the image to add variant to
- `variant_name` (string, required): Name of the variant
- `tag_suffix` (string, required): Suffix to append to tags (e.g., -slim)
- `versions` (object, optional): Version overrides for this variant
- `build_args` (object, optional): Build args for this variant

**Returns:** Confirmation message.

### search_documentation

Search ContainerHive documentation by query text.

**Parameters:**

- `query` (string, required): Search query text
- `limit` (integer, optional): Max results to return (default 10)

**Returns:** Array of results with title, path, and excerpt.

### get_documentation

Fetch full documentation page content by path.

**Parameters:**

- `path` (string, required): Path to the documentation page (e.g., `index.html`, `usage/mcp.html`)

**Returns:** Object with title, url (GitHub link), and full content (markdown).

## Resources

### image://schema

JSON schema for image.yml configuration files.

### project://schema

JSON schema for hive.yml configuration files.

### project://config

The project's hive.yml configuration.

### image://{name}

Image configuration file for a specific image. Use `{name}` as a URI template parameter.

## MCP Client Configuration

=== "OpenCode"

    **CLI:**
    ```bash
    opencode mcp add containerhive -- ch mcp --project ${workspace}
    ```

    **Or JSON** in `~/.opencode/mcp.json`:
    ```json title="~/.opencode/mcp.json"
    {
      "mcpServers": {
        "containerhive": {
          "command": "ch",
          "args": [
            "mcp",
            "--project",
            "${workspace}"
          ]
        }
      }
    }
    ```

=== "Claude Code"

    **CLI:**
    ```bash
    claude mcp add containerhive -- ch mcp --project ${workspace}
    ```

    **Or JSON** at user scope in `~/.claude.json`:
    ```json title="~/.claude.json"
    {
      "mcpServers": {
        "containerhive": {
          "command": "ch",
          "args": [
            "mcp",
            "--project",
            "${workspace}"
          ]
        }
      }
    }
    ```

    **Or project-scoped** in project root:
    ```json title=".mcp.json"
    {
      "mcpServers": {
        "containerhive": {
          "command": "ch",
          "args": [
            "mcp",
            "--project",
            "${workspace}"
          ]
        }
      }
    }
    ```

=== "Claude Desktop"

    Add to `~/Library/Application Support/Claude/settings.json`:

    ```json title="~/Library/Application Support/Claude/settings.json"
    {
      "mcpServers": {
        "containerhive": {
          "command": "ch",
          "args": [
            "mcp",
            "--project",
            "/path/to/project"
          ]
        }
      }
    }
    ```

=== "Cursor"

    Add to `~/.cursor/mcp.json`:

    ```json title="~/.cursor/mcp.json"
    {
      "mcpServers": {
        "containerhive": {
          "command": "ch",
          "args": [
            "mcp",
            "--project",
            "${workspace}"
          ]
        }
      }
    }
    ```