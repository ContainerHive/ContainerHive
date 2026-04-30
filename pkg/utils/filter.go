package utils

import (
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/build"
)

// ParseFilters converts positional CLI arguments into build filters.
// Each argument can be "imageName" or "imageName:tagName".
func ParseFilters(args []string) []build.Filter {
	var filters []build.Filter
	for _, arg := range args {
		f := build.Filter{}
		if idx := strings.IndexByte(arg, ':'); idx >= 0 {
			f.ImageName = arg[:idx]
			f.TagName = arg[idx+1:]
		} else {
			f.ImageName = arg
		}
		filters = append(filters, f)
	}
	return filters
}

// MatchesFilter checks if an image:tag matches the given filters.
// Empty filters match everything.
func MatchesFilter(filters []build.Filter, imageName, tagName string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f.ImageName != "" && f.ImageName != imageName {
			continue
		}
		if f.TagName != "" && f.TagName != tagName {
			continue
		}
		return true
	}
	return false
}
