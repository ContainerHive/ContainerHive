package mcp

const imageSchemaJSON = `{
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "description": "Description of the image"
    },
    "tags": {
      "type": [
        "null",
        "array"
      ],
      "items": {
        "type": [
          "null",
          "object"
        ],
        "properties": {
          "name": {
            "type": "string",
            "description": "Name of the tag"
          },
          "versions": {
            "type": "object",
            "description": "Versions to use for this tag",
            "additionalProperties": {
              "type": "string"
            }
          },
          "build_args": {
            "type": "object",
            "description": "Build args to specify for this tag",
            "additionalProperties": {
              "type": "string"
            }
          }
        },
        "required": [
          "name"
        ],
        "additionalProperties": false
      },
      "description": "Tags to create for this image"
    },
    "variants": {
      "type": [
        "null",
        "array"
      ],
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "Name of the variant"
          },
          "tag_suffix": {
            "type": "string",
            "description": "Suffix to append to the tag name for this variant"
          },
          "versions": {
            "type": "object",
            "description": "Versions to use for this variant",
            "additionalProperties": {
              "type": "string"
            }
          },
          "build_args": {
            "type": "object",
            "description": "Build args to add for this variant",
            "additionalProperties": {
              "type": "string"
            }
          },
          "platforms": {
            "type": [
              "null",
              "array"
            ],
            "items": {
              "type": "string"
            },
            "description": "Target platforms for this variant (e.g. linux/amd64)"
          },
          "report": {
            "type": "object",
            "properties": {
              "icon": {
                "type": [
                  "null",
                  "string"
                ],
                "description": "Icon slug for devicon (e.g. go-original)"
              }
            },
            "description": "Report metadata",
            "required": [
              "icon"
            ],
            "additionalProperties": false
          }
        },
        "required": [
          "name",
          "tag_suffix"
        ],
        "additionalProperties": false
      },
      "description": "Variants to create for this image"
    },
    "versions": {
      "type": "object",
      "description": "Versions to use for this image",
      "additionalProperties": {
        "type": "string"
      }
    },
    "build_args": {
      "type": "object",
      "description": "Build args to add for this image",
      "additionalProperties": {
        "type": "string"
      }
    },
    "secrets": {
      "type": "object",
      "description": "Secrets to resolve for this image",
      "additionalProperties": {
        "type": "object",
        "properties": {
          "source": {
            "type": "string",
            "description": "Source type of the secret (env, plain). If omitted, auto-detected from value."
          },
          "value": {
            "type": "string",
            "description": "Value of the secret (env var name or plain text)"
          }
        },
        "required": [
          "value"
        ],
        "additionalProperties": false
      }
    },
    "depends_on": {
      "type": [
        "null",
        "array"
      ],
      "items": {
        "type": "string"
      },
      "description": "Names of other images in this project that must be built before this image"
    },
    "platforms": {
      "type": [
        "null",
        "array"
      ],
      "items": {
        "type": "string"
      },
      "description": "Target platforms for this image (e.g. linux/amd64)"
    },
    "latest_alias": {
      "type": [
        "null",
        "object"
      ],
      "properties": {
        "tag": {
          "type": "string",
          "description": "Alias tag name to assign to the highest semantic version (e.g. latest, stable),required"
        },
        "on_missing": {
          "type": "string",
          "description": "Behaviour when no semantic tags are found: error (default), warning, silent"
        }
      },
      "description": "Configure an alias pointing to the highest semantic version tag",
      "required": [
        "tag"
      ],
      "additionalProperties": false
    },
    "report": {
      "type": "object",
      "properties": {
        "icon": {
          "type": [
            "null",
            "string"
          ],
          "description": "Icon slug for devicon (e.g. go-original)"
        }
      },
      "description": "Report metadata",
      "required": [
        "icon"
      ],
      "additionalProperties": false
    }
  },
  "$id": "https://container-hive.timo-reymann.de/schemas/image.schema.json",
  "title": "Image definition",
  "description": "Image definition configuration schema for ContainerHive.",
  "required": [
    "tags"
  ],
  "additionalProperties": false
}`

const hiveSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "buildkit": {
      "type": "object",
      "properties": {
        "address": {
          "type": "string",
          "description": "BuildKit daemon address (e.g. tcp://127.0.0.1:8502)"
        }
      },
      "required": [
        "address"
      ],
      "additionalProperties": false
    },
    "cache": {
      "type": "object",
      "properties": {
        "type": {
          "type": "string",
          "description": "Cache type (s3, registry),required"
        },
        "endpoint": {
          "type": "string",
          "description": "S3 endpoint URL"
        },
        "bucket": {
          "type": "string",
          "description": "S3 bucket name"
        },
        "region": {
          "type": "string",
          "description": "S3 region"
        },
        "access_key_id": {
          "type": "string",
          "description": "S3 access key ID"
        },
        "secret_access_key": {
          "type": "string",
          "description": "S3 secret access key"
        },
        "use_path_style": {
          "type": "boolean",
          "description": "Use path-style S3 URLs"
        },
        "ref": {
          "type": "string",
          "description": "Registry cache ref (e.g. registry:5000/cache)"
        },
        "insecure": {
          "type": "boolean",
          "description": "Allow insecure registry connections"
        }
      },
      "required": [
        "type"
      ],
      "additionalProperties": false
    },
    "registry": {
      "type": "object",
      "properties": {
        "address": {
          "type": "string",
          "description": "Container registry address"
        }
      },
      "required": [
        "address"
      ],
      "additionalProperties": false
    },
    "platforms": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Default target platforms for all images (e.g. linux/amd64)"
    },
    "template_options": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Custom template variables available via the option function in CI and custom templates"
    }
  },
  "additionalProperties": false
}`
