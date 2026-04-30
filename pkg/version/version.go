package version

import "github.com/ContainerHive/ContainerHive/internal/buildinfo"

func Get() string {
	return buildinfo.Version
}
