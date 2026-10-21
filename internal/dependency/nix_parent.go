package dependency

import "github.com/ContainerHive/ContainerHive/pkg/model"

// nixParentRefs returns only nix.parent values of Nix-backend builds; the
// Dockerfile backend ignores them. Variants without a nix block inherit the image's.
func nixParentRefs(img *model.Image) []string {
	var refs []string
	if img.BuildBackend == model.BuildBackendNix && img.Nix.Parent != "" {
		refs = append(refs, img.Nix.Parent)
	}
	for _, variant := range img.Variants {
		backend := variant.BuildBackend
		if backend == "" {
			backend = img.BuildBackend
		}
		if backend != model.BuildBackendNix {
			continue
		}
		parent := img.Nix.Parent
		if variant.Nix != nil {
			parent = variant.Nix.Parent
		}
		if parent != "" {
			refs = append(refs, parent)
		}
	}
	return refs
}
