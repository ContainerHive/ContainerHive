package report

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

type RegistryScanner struct {
	registryAddr string
	project      *model.ContainerHiveProject
	auth         authn.Authenticator
}

func NewRegistryScanner(registryAddr string, project *model.ContainerHiveProject) *RegistryScanner {
	return &RegistryScanner{
		registryAddr: registryAddr,
		project:      project,
	}
}

func (s *RegistryScanner) Scan(ctx context.Context) ([]ImageReport, error) {
	auth, err := s.getAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry auth: %w", err)
	}
	s.auth = auth

	var images []ImageReport

	for imageName := range s.project.ImagesByName {
		imageReport, err := s.scanImage(ctx, imageName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image %s: %w", imageName, err)
		}
		if imageReport.Name != "" {
			images = append(images, imageReport)
		}
	}

	return images, nil
}

func (s *RegistryScanner) getAuth() (authn.Authenticator, error) {
	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		return nil, fmt.Errorf("failed to load docker config: %w", err)
	}

	reg, err := name.NewRegistry(s.registryAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	serverAddress := reg.Name()
	if serverAddress == name.DefaultRegistry {
		serverAddress = "https://index.docker.io/v1/"
	}

	creds := cf.GetCredentialsStore(serverAddress)
	authConfig, err := creds.Get(serverAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	return authn.FromConfig(authn.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}), nil
}

func (s *RegistryScanner) scanImage(ctx context.Context, imageName string) (ImageReport, error) {
	modelImages := s.project.ImagesByName[imageName]
	if len(modelImages) == 0 {
		return ImageReport{}, nil
	}

	imgModel := modelImages[0]

	var tagReports []TagReport
	for _, tagDef := range imgModel.Tags {
		var platformReports []PlatformReport

		for _, plat := range imgModel.Platforms {
			digest, size := s.fetchManifestDigest(ctx, imageName, tagDef.Name, plat)

			platformReports = append(platformReports, PlatformReport{
				Platform:  plat,
				Digest:    digest,
				Size:      size,
				HasSBOM:   false,
				BuildArgs: tagDef.BuildArgs,
			})
		}

		tagReports = append(tagReports, TagReport{
			Name:      tagDef.Name,
			Platforms: platformReports,
		})
	}

	var variantReports []VariantReport
	for variantName, variantDef := range imgModel.Variants {
		var variantTagReports []TagReport

		for _, baseTag := range imgModel.Tags {
			var platformReports []PlatformReport

			for _, plat := range variantDef.Platforms {
				fullTagName := baseTag.Name + variantDef.TagSuffix
				digest, size := s.fetchManifestDigest(ctx, imageName, fullTagName, plat)

				mergedBuildArgs := mergeBuildArgs(baseTag.BuildArgs, variantDef.BuildArgs)

				platformReports = append(platformReports, PlatformReport{
					Platform:  plat,
					Digest:    digest,
					Size:      size,
					HasSBOM:   false,
					BuildArgs: mergedBuildArgs,
				})
			}

			variantTagReports = append(variantTagReports, TagReport{
				Name:      baseTag.Name + variantDef.TagSuffix,
				Platforms: platformReports,
			})
		}

		variantReports = append(variantReports, VariantReport{
			Name:      variantName,
			TagSuffix: variantDef.TagSuffix,
			Tags:      variantTagReports,
		})
	}

	return ImageReport{
		Name:     imageName,
		Versions: imgModel.Versions,
		Tags:     tagReports,
		Variants: variantReports,
	}, nil
}

func mergeBuildArgs(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func (s *RegistryScanner) fetchManifestDigest(ctx context.Context, imageName, tagName, platform string) (string, int64) {
	fullTag := fmt.Sprintf("%s/%s:%s-%s", s.registryAddr, imageName, tagName, platform)

	ref, err := name.NewTag(fullTag)
	if err != nil {
		return "", 0
	}

	img, err := remote.Image(ref, remote.WithAuth(s.auth), remote.WithContext(ctx))
	if err != nil {
		return "", 0
	}

	digest, _ := img.Digest()
	size, _ := img.Size()

	return digest.String(), size
}

func (s *RegistryScanner) Source() string {
	return "registry"
}
