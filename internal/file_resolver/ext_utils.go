package file_resolver

import "path/filepath"

// RemoveTemplateExt strips a recognized template extension (e.g. ".gotpl") from the filename if present.
func RemoveTemplateExt(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) < 1 {
		return filename
	}

	if _, ok := processorMapping[ext[1:]]; ok {
		return filename[:len(filename)-len(ext)]
	}

	return filename
}
