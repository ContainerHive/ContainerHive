package file_resolver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/timo-reymann/ContainerHive/internal/file_resolver/templating"
)

// CopyAndRenderFile copies src to target, rendering it through a template processor if the file extension matches a supported templating engine.
func CopyAndRenderFile(tmplCtx *templating.TemplateContext, src, target string) error {
	ext, _ := strings.CutPrefix(filepath.Ext(src), ".")

	if len(ext) < 2 {
		_, err := fileutils.CopyFile(src, target)
		return err
	}

	processor, ok := processorMapping[ext]
	if !ok {
		_, err := fileutils.CopyFile(src, target)
		return err

	}

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	rendered, err := processor.Process(tmplCtx, src, content)
	if err != nil {
		return err
	}
	return os.WriteFile(target, rendered, 0644)
}
