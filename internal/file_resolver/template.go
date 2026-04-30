package file_resolver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver/templating"
	"github.com/docker/docker/pkg/fileutils"
)

// ReadAndRenderFile reads src and renders it through a template processor if the file extension matches a supported templating engine. Returns raw content otherwise.
func ReadAndRenderFile(tmplCtx *templating.TemplateContext, src string) ([]byte, error) {
	content, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}

	ext, _ := strings.CutPrefix(filepath.Ext(src), ".")
	if len(ext) < 2 {
		return content, nil
	}

	processor, ok := processorMapping[ext]
	if !ok {
		return content, nil
	}

	return processor.Process(tmplCtx, src, content)
}

// CopyAndRenderFile copies src to target, rendering it through a template processor if the file extension matches a supported templating engine.
func CopyAndRenderFile(tmplCtx *templating.TemplateContext, src, target string) error {
	ext, _ := strings.CutPrefix(filepath.Ext(src), ".")
	if len(ext) < 2 {
		_, err := fileutils.CopyFile(src, target)
		return err
	}

	if _, ok := processorMapping[ext]; !ok {
		_, err := fileutils.CopyFile(src, target)
		return err
	}

	rendered, err := ReadAndRenderFile(tmplCtx, src)
	if err != nil {
		return err
	}
	return os.WriteFile(target, rendered, 0644)
}
