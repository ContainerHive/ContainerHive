package templating

import (
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// TemplateContext holds the data available to template processors during rendering.
type TemplateContext struct {
	Versions  model.Versions
	BuildArgs model.BuildArgs
	ImageName string
}

// Processor renders template content using the given context and source path.
type Processor interface {
	// Process renders the given content bytes using the template context.
	Process(*TemplateContext, string, []byte) ([]byte, error)
}
