package lint

import (
	"bytes"
	"fmt"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// SubstituteHiveParent replaces every model.HiveParentPlaceholder token in
// content with parentRef. Returns content unchanged when the placeholder is
// absent or parentRef is empty.
func SubstituteHiveParent(content []byte, parentRef string) []byte {
	if parentRef == "" || !bytes.Contains(content, []byte(model.HiveParentPlaceholder)) {
		return content
	}
	return bytes.ReplaceAll(content, []byte(model.HiveParentPlaceholder), []byte(parentRef))
}

// BuildHiveParentRef returns the __hive__/<image>:<tag> reference that
// rendering would substitute for the placeholder in this image's variants.
// It mirrors pkg/rendering.replaceHiveParent's format so source linting sees
// the same FROM line as a built image would.
func BuildHiveParentRef(img *model.Image) string {
	return fmt.Sprintf("__hive__/%s:%s", img.Name, PickReferenceTag(img.Tags))
}

// PickReferenceTag returns a deterministic tag name from the image's tag set.
// Any tag suffices for hadolint (it only checks that FROM has *a* tag), but
// picking the lexicographically first one keeps lint output stable across
// runs.
func PickReferenceTag(tags map[string]*model.Tag) string {
	if len(tags) == 0 {
		return "hive-parent"
	}
	first := ""
	for name := range tags {
		if first == "" || name < first {
			first = name
		}
	}
	return first
}
