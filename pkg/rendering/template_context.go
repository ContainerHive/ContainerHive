package rendering

import (
	"github.com/ContainerHive/ContainerHive/internal/buildconfig_resolver"
	"github.com/ContainerHive/ContainerHive/internal/file_resolver/templating"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func newTemplateContext(image *model.Image, values *buildconfig_resolver.ResolvedBuildValues) *templating.TemplateContext {
	return &templating.TemplateContext{
		ImageName: image.Name,
		Versions:  values.Versions,
		BuildArgs: values.BuildArgs,
	}
}
