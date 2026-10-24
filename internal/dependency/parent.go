package dependency

import "github.com/ContainerHive/ContainerHive/pkg/model"

// backendParentRefs returns only the parent values of Nix and Chisel backend
// builds; the Dockerfile backend ignores them. Variants without a config
// block inherit the image level parent.
func backendParentRefs(img *model.Image) []string {
	var refs []string
	if img.BuildBackend == model.BuildBackendNix && img.Nix.Parent != "" {
		refs = append(refs, img.Nix.Parent)
	}
	if img.BuildBackend == model.BuildBackendChisel && img.Chisel.Parent != "" {
		refs = append(refs, img.Chisel.Parent)
	}
	for _, variant := range img.Variants {
		backend := variant.BuildBackend
		if backend == "" {
			backend = img.BuildBackend
		}
		switch backend {
		case model.BuildBackendNix:
			parent := img.Nix.Parent
			if variant.Nix != nil {
				parent = variant.Nix.Parent
			}
			if parent != "" {
				refs = append(refs, parent)
			}
		case model.BuildBackendChisel:
			parent := img.Chisel.Parent
			if variant.Chisel != nil {
				parent = variant.Chisel.Parent
			}
			if parent != "" {
				refs = append(refs, parent)
			}
		}
	}
	return refs
}
