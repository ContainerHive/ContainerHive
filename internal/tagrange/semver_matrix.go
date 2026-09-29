package tagrange

import (
	"fmt"
	"strconv"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// generatorFunc expands a named generator's params into an equivalent
// explicit source/select/filter triple, so the resolver has exactly one
// code path regardless of whether a range used source: or generator:.
type generatorFunc func(params map[string]string) (*model.SourceConfig, *model.SelectConfig, *model.FilterConfig, error)

var generators = map[string]generatorFunc{
	"semver-matrix": semverMatrixGenerator,
}

func expandGenerator(name string, params map[string]string) (*model.SourceConfig, *model.SelectConfig, *model.FilterConfig, error) {
	gen, ok := generators[name]
	if !ok {
		known := make([]string, 0, len(generators))
		for k := range generators {
			known = append(known, k)
		}
		return nil, nil, nil, fmt.Errorf("unknown generator %q (known: %v)", name, known)
	}
	return gen(params)
}

// semverMatrixParams are the accepted keys for the semver-matrix generator.
// Any other key is a hard error.
var semverMatrixParams = map[string]struct{}{
	"source": {}, "image": {}, "repo": {}, "url": {}, "transform": {},
	"majors": {}, "minors": {}, "patches": {},
}

// semverMatrixGenerator expands params into a source plus a select block
// with the requested major/minor/patch window and exclude_prerelease: true.
func semverMatrixGenerator(params map[string]string) (*model.SourceConfig, *model.SelectConfig, *model.FilterConfig, error) {
	for key := range params {
		if _, ok := semverMatrixParams[key]; !ok {
			return nil, nil, nil, fmt.Errorf("semver-matrix: unknown param %q (accepted: source, image, repo, url, transform, majors, minors, patches)", key)
		}
	}

	sourceType := params["source"]
	if sourceType == "" {
		return nil, nil, nil, fmt.Errorf("semver-matrix: param \"source\" is required")
	}

	majors, err := positiveIntParam(params, "majors", 1)
	if err != nil {
		return nil, nil, nil, err
	}
	minors, err := positiveIntParam(params, "minors", 1)
	if err != nil {
		return nil, nil, nil, err
	}
	patches, err := positiveIntParam(params, "patches", 1)
	if err != nil {
		return nil, nil, nil, err
	}

	cfg := &model.SourceConfig{
		Type:      sourceType,
		Image:     params["image"],
		Repo:      params["repo"],
		URL:       params["url"],
		Transform: params["transform"],
	}
	sel := &model.SelectConfig{
		Major: &model.LevelSelect{Last: majors},
		Minor: &model.LevelSelect{Last: minors},
		Patch: &model.LevelSelect{Last: patches},
	}
	excludePrerelease := true
	filter := &model.FilterConfig{ExcludePrerelease: &excludePrerelease}
	return cfg, sel, filter, nil
}

func positiveIntParam(params map[string]string, key string, def int) (int, error) {
	v, ok := params[key]
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("semver-matrix: param %q must be a positive integer, got %q", key, v)
	}
	return n, nil
}
