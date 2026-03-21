package file_resolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/internal/file_resolver/templating"
)

const TemplateExtensionGoTemplate = "gotpl"

var supportedTemplateExtensions = []string{
	TemplateExtensionGoTemplate,
}

var processorMapping = map[string]templating.Processor{
	TemplateExtensionGoTemplate: &templating.GoTemplateTemplatingProcessor{},
}

var NoFileCandidatesErr = errors.New("no file candidates found")

func GetFileCandidates(baseName string, extensions ...string) []string {
	extLen := len(extensions)
	var possibleNames []string

	if extLen == 0 {
		possibleNames = make([]string, len(supportedTemplateExtensions)+1)
		possibleNames[0] = baseName
		for idx, tmplExt := range supportedTemplateExtensions {
			possibleNames[idx+1] = fmt.Sprintf("%s.%s", baseName, tmplExt)
		}
	} else {
		possibleNames = make([]string, extLen*(len(supportedTemplateExtensions)+1))
		idx := 0
		for _, ext := range extensions {
			possibleNames[idx] = fmt.Sprintf("%s.%s", baseName, ext)
			idx++
			for _, tmplExt := range supportedTemplateExtensions {
				possibleNames[idx] = fmt.Sprintf("%s.%s.%s", baseName, ext, tmplExt)
				idx++
			}
		}
	}

	return possibleNames
}

func ResolveFirstExistingFile(root string, candidates ...string) (string, error) {
	for _, candidate := range candidates {
		candidatePath := filepath.Join(root, candidate)
		if stat, err := os.Stat(candidatePath); err == nil && !stat.IsDir() {
			return candidatePath, nil
		}
	}
	return "", NoFileCandidatesErr
}
