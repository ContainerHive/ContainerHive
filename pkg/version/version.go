package version

import "github.com/timo-reymann/ContainerHive/internal/buildinfo"

func Get() string {
	return buildinfo.Version
}
