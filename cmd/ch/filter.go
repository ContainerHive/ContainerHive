package main

import (
	"strings"

	"github.com/timo-reymann/ContainerHive/pkg/build"
)

// parseFilters converts positional CLI arguments into build filters.
// Each argument can be "imageName" or "imageName:tagName".
func parseFilters(args []string) []build.Filter {
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
