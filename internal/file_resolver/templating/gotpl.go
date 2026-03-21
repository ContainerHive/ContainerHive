package templating

import (
	pkgtemplating "github.com/timo-reymann/ContainerHive/pkg/templating"
)

// GoTemplateTemplatingProcessor implements Processor using Go's text/template engine.
type GoTemplateTemplatingProcessor struct{}

func (g *GoTemplateTemplatingProcessor) Process(tplCtx *TemplateContext, path string, content []byte) ([]byte, error) {
	return pkgtemplating.RenderString(path, string(content), tplCtx)
}
