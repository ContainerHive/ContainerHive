package dependency

import "github.com/ContainerHive/ContainerHive/pkg/model"

// backendParentRefs returns only the parent values of Nix and Chisel backend
// builds; the Dockerfile backend ignores them. Variants without a config
// block inherit the image level parent.
func backendParentRefs(img *model.Image) []string {
	var refs []string
	if parent := imageParentRef(img); parent != "" {
		refs = append(refs, parent)
	}
	for _, variant := range img.Variants {
		if parent := variantParentRef(img, variant); parent != "" {
			refs = append(refs, parent)
		}
	}
	return refs
}

// imageParentRef returns the backend parent of the image itself, or an empty
// string for the Dockerfile backend.
func imageParentRef(img *model.Image) string {
	switch img.BuildBackend {
	case model.BuildBackendNix:
		return img.Nix.Parent
	case model.BuildBackendChisel:
		return img.Chisel.Parent
	}
	return ""
}

// variantParentRef returns the backend parent of a variant, falling back to
// the image level backend and parent when the variant sets none.
func variantParentRef(img *model.Image, variant *model.ImageVariant) string {
	backend := variant.BuildBackend
	if backend == "" {
		backend = img.BuildBackend
	}
	switch backend {
	case model.BuildBackendNix:
		if variant.Nix != nil {
			return variant.Nix.Parent
		}
		return img.Nix.Parent
	case model.BuildBackendChisel:
		if variant.Chisel != nil {
			return variant.Chisel.Parent
		}
		return img.Chisel.Parent
	}
	return ""
}
