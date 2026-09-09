package source

import (
	"context"
	"fmt"

	"github.com/ContainerHive/ContainerHive/internal/jsonata"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// JSONSource fetches an arbitrary JSON document and extracts versions via a
// JSONata transform, for upstream feeds like nodejs.org/dist/index.json
// that have no dedicated source.
type JSONSource struct{}

func (JSONSource) Name() string { return "json" }

func (JSONSource) Validate(cfg *model.SourceConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("source type %q requires url", "json")
	}
	if cfg.Image != "" || cfg.Repo != "" || cfg.Kind != "" {
		return fmt.Errorf("source type %q does not accept image/repo/kind", "json")
	}
	return nil
}

func (JSONSource) Descriptor(cfg *model.SourceConfig) string {
	return "json (" + cfg.URL + ")"
}

func (JSONSource) CacheKeyParts(cfg *model.SourceConfig) []string {
	return []string{"json", cfg.URL, cfg.Transform}
}

func (JSONSource) Fetch(ctx context.Context, cfg *model.SourceConfig) ([]Version, error) {
	var doc any
	if err := getJSON(ctx, cfg.URL, cfg.Headers, &doc); err != nil {
		return nil, fmt.Errorf("json source %q: %w", cfg.URL, err)
	}

	result := doc
	if cfg.Transform != "" {
		transformed, err := jsonata.EvalTransform(cfg.Transform, doc)
		if err != nil {
			return nil, fmt.Errorf("json source %q: %w", cfg.URL, err)
		}
		result = transformed
	}

	return parseVersions(result)
}

// parseVersions accepts the transform's output in the two documented
// shapes: an array of {"version": ..., ...extras} objects, or a plain array
// of version strings.
func parseVersions(result any) ([]Version, error) {
	items, ok := result.([]any)
	if !ok {
		// A transform that reduces to a single object (e.g. "$[0]") is
		// treated as a one-element result rather than an error.
		items = []any{result}
	}

	versions := make([]Version, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			versions = append(versions, Version{Raw: v})
		case map[string]any:
			ver, err := versionFromObject(v)
			if err != nil {
				return nil, err
			}
			versions = append(versions, ver)
		default:
			return nil, fmt.Errorf("transform result contains an unsupported element type %T", item)
		}
	}
	return versions, nil
}

func versionFromObject(obj map[string]any) (Version, error) {
	rawAny, ok := obj["version"]
	if !ok {
		return Version{}, fmt.Errorf(`transform result object is missing a "version" field: %v`, obj)
	}
	raw, ok := rawAny.(string)
	if !ok {
		return Version{}, fmt.Errorf(`transform result "version" field must be a string, got %T`, rawAny)
	}

	extra := make(map[string]string)
	for key, val := range obj {
		if key == "version" {
			continue
		}
		if s, ok := stringify(val); ok {
			extra[key] = s
		}
	}

	prerelease, _ := obj["prerelease"].(bool)
	return Version{Raw: raw, Prerelease: prerelease, Extra: extra}, nil
}

// stringify converts a JSON scalar to its string form for .extra.<key>.
// Nested objects/arrays are skipped (not an error - extras are best-effort
// metadata, not required fields).
func stringify(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case bool:
		if val {
			return "true", true
		}
		return "false", true
	case float64:
		return fmt.Sprintf("%v", val), true
	default:
		return "", false
	}
}
