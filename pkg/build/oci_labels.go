package build

import (
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

const (
	ociLabelTitle         = "org.opencontainers.image.title"
	ociLabelRefName       = "org.opencontainers.image.ref.name"
	ociLabelVersion       = "org.opencontainers.image.version"
	ociLabelCreated       = "org.opencontainers.image.created"
	ociLabelDescription   = "org.opencontainers.image.description"
	ociLabelVendor        = "org.opencontainers.image.vendor"
	ociLabelAuthors       = "org.opencontainers.image.authors"
	ociLabelURL           = "org.opencontainers.image.url"
	ociLabelDocumentation = "org.opencontainers.image.documentation"
	ociLabelRevision      = "org.opencontainers.image.revision"
	ociLabelSource        = "org.opencontainers.image.source"
)

// OCILabelArgs are inputs to BuildOCILabels.
type OCILabelArgs struct {
	// ImageName is the bare image name without tag (e.g. "python").
	ImageName string
	// Tag is the full tag including any variant suffix (e.g. "3.12-alpine").
	Tag string
	// Description is the image description; emitted only when non-empty.
	Description string
	// ProjectRoot is the directory used to probe for a git repository. Empty
	// disables git-derived labels.
	ProjectRoot string
	// Project carries project-level label values from hive.yml.
	Project *model.LabelsConfig
	// ImageLabels are per-image custom labels declared in image.yml.
	ImageLabels map[string]string
	// TagLabels are per-tag custom labels; override image labels.
	TagLabels map[string]string
	// VariantLabels are per-variant custom labels; override tag and image labels.
	VariantLabels map[string]string
}

// BuildOCILabels assembles the OCI annotation labels for a single image build.
// Always-on labels: title, ref.name, version, created. Description, project,
// and git-derived labels are emitted only when their inputs are present.
func BuildOCILabels(a OCILabelArgs) map[string]string {
	labels := map[string]string{}

	// Custom labels merge in precedence order (least → most specific): project,
	// image, tag, variant. Standard auto-derived keys are written afterwards
	// and override any custom collisions.
	if a.Project != nil {
		for k, v := range a.Project.Custom {
			labels[k] = v
		}
	}
	for k, v := range a.ImageLabels {
		labels[k] = v
	}
	for k, v := range a.TagLabels {
		labels[k] = v
	}
	for k, v := range a.VariantLabels {
		labels[k] = v
	}

	labels[ociLabelTitle] = a.ImageName
	labels[ociLabelRefName] = a.ImageName
	labels[ociLabelVersion] = a.Tag
	labels[ociLabelCreated] = time.Now().UTC().Format(time.RFC3339)

	if a.Description != "" {
		labels[ociLabelDescription] = a.Description
	}
	if a.Project != nil {
		if a.Project.Vendor != "" {
			labels[ociLabelVendor] = a.Project.Vendor
		}
		if a.Project.Authors != "" {
			labels[ociLabelAuthors] = a.Project.Authors
		}
		if a.Project.Url != "" {
			labels[ociLabelURL] = applyLabelPlaceholders(a.Project.Url, a.ImageName, a.Tag)
		}
		if a.Project.Documentation != "" {
			labels[ociLabelDocumentation] = applyLabelPlaceholders(a.Project.Documentation, a.ImageName, a.Tag)
		}
	}
	revision, source := gitInfo(a.ProjectRoot)
	if revision != "" {
		labels[ociLabelRevision] = revision
	}
	if source != "" {
		labels[ociLabelSource] = source
	}
	return labels
}

func applyLabelPlaceholders(s, image, tag string) string {
	s = strings.ReplaceAll(s, "%image%", image)
	s = strings.ReplaceAll(s, "%tag%", tag)
	return s
}

// gitInfo returns the HEAD commit hash and origin remote URL for the repo
// containing repoRoot. Any failure (no repo, no commits, no origin) yields
// empty strings so the caller can simply skip the label.
func gitInfo(repoRoot string) (revision, remoteURL string) {
	if repoRoot == "" {
		return "", ""
	}
	repo, err := git.PlainOpenWithOptions(repoRoot, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return "", ""
	}
	if head, err := repo.Head(); err == nil {
		revision = head.Hash().String()
	}
	if remote, err := repo.Remote("origin"); err == nil {
		if cfg := remote.Config(); cfg != nil && len(cfg.URLs) > 0 {
			remoteURL = cfg.URLs[0]
		}
	}
	return revision, remoteURL
}
