package tagrange

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/templating"
)

// ociTagPattern is the OCI/Docker tag grammar. Upstream JSON routinely
// contains values that don't fit it (build metadata like "+build.1", "~",
// empty strings), so every rendered tag name is validated against it.
var ociTagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)

// templateData builds the map handed to text/template for one candidate.
// text/template's field lookup on a struct is by exact exported name, so
// {{.major}} cannot resolve against a struct field Major — the data must be
// a map with lowercase keys, matching the documented template syntax.
func templateData(c candidate) map[string]any {
	major, minor, patch := c.ver.Major(), c.ver.Minor(), c.ver.Patch()
	return map[string]any{
		"major":      fmt.Sprintf("%d", major),
		"minor":      fmt.Sprintf("%d", minor),
		"patch":      fmt.Sprintf("%d", patch),
		"full":       c.ver.String(),
		"raw":        c.raw,
		"prerelease": c.ver.Prerelease(),
		"extra":      c.extra,
	}
}

// renderTagName renders the tag_name template for one candidate and
// validates the result against the OCI tag grammar.
func renderTagName(rangeLabel, tmpl string, c candidate) (string, error) {
	out, err := templating.RenderString("tag_range["+rangeLabel+"]:tag_name", tmpl, templateData(c))
	if err != nil {
		return "", fmt.Errorf("tag_name template failed for version %q: %w", c.raw, err)
	}
	name := string(out)
	if !ociTagPattern.MatchString(name) {
		return "", fmt.Errorf("tag_name template rendered %q for version %q, which is not a valid OCI tag (expected %s)", name, c.raw, ociTagPattern.String())
	}
	return name, nil
}

// missingMapKey is what text/template renders for a map field access whose
// key does not exist (Go templates render an invalid/missing lookup this
// way rather than an empty string).
const missingMapKey = "<no value>"

// renderValues renders a map of Go templates (versions, build_args, or
// labels) for one candidate. A rendered value referencing a missing
// .extra.<key> is treated as an error, since it usually means a typo'd
// extra key would otherwise silently ship as a build arg.
func renderValues(rangeLabel, kind string, templates map[string]string, c candidate) (map[string]string, error) {
	if len(templates) == 0 {
		return nil, nil
	}
	data := templateData(c)
	result := make(map[string]string, len(templates))
	for key, tmpl := range templates {
		out, err := templating.RenderString(fmt.Sprintf("tag_range[%s]:%s.%s", rangeLabel, kind, key), tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("%s %q template failed for version %q: %w", kind, key, c.raw, err)
		}
		rendered := string(out)
		if strings.Contains(rendered, missingMapKey) && strings.Contains(tmpl, ".extra.") {
			return nil, fmt.Errorf("%s %q rendered %q for version %q (template %q references a missing .extra. key)", kind, key, rendered, c.raw, tmpl)
		}
		result[key] = rendered
	}
	return result, nil
}
