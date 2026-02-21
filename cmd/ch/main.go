package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/timo-reymann/ContainerHive/internal/buildconfig_resolver"
	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/cache"
	"github.com/timo-reymann/ContainerHive/internal/container_structure_test"
	"github.com/timo-reymann/ContainerHive/internal/dependency"
	"github.com/timo-reymann/ContainerHive/internal/docker"
	"github.com/timo-reymann/ContainerHive/internal/gcr"
	"github.com/timo-reymann/ContainerHive/internal/registry"
	"github.com/timo-reymann/ContainerHive/internal/syft"
	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/rendering"
)

const (
	// Matches hack/docker-compose.yml buildkitd service
	buildkitAddr = "tcp://127.0.0.1:8502"

	// Matches hack/garage/init.sh S3 cache configuration
	// Note: Use docker-compose service name 'garage' since buildkitd runs in container
	s3Endpoint  = "http://127.0.0.1:39505"
	s3Bucket    = "buildkit-cache"
	s3Region    = "garage"
	s3AccessKey = "GK31337cafe000000000000000"
	s3SecretKey = "1337cafe0000000000000000000000000000000000000000000000000000dead"

	imageName = "ch-smoke-test:latest"
)

var platform = "linux/" + runtime.GOARCH

// newProgressWriter returns a buildkit status handler that displays build progress.
func newProgressWriter() func(chan *client.SolveStatus) error {
	return func(ch chan *client.SolveStatus) error {
		// TODO for production support writing trace
		d, err := progressui.NewDisplay(os.Stdout, progressui.TtyMode)
		if err != nil {
			d, _ = progressui.NewDisplay(os.Stdout, progressui.PlainMode)
		}
		_, err = d.UpdateFrom(context.TODO(), ch)
		return err
	}
}

// patchHiveRefs rewrites __hive__/ references in a Dockerfile for registry use.
// Returns the patched file path and a cleanup function.
func patchHiveRefs(dockerfilePath, registryAddr string) (string, func()) {
	patched := dockerfilePath + ".patched"
	if err := build_context.RewriteHiveRefs(dockerfilePath, patched, registryAddr); err != nil {
		log.Fatalf("Failed to rewrite hive refs for %s: %v", dockerfilePath, err)
	}
	return patched, func() { os.Remove(patched) }
}

// tarFilePath returns the OCI tar output path inside the rendered dist directory for a given image tag.
func tarFilePath(distPath, name, tag string) string {
	return filepath.Join(distPath, name, tag, "image.tar")
}

// collectTestDefinitions finds test YAML files in a rendered dist directory's tests/ subfolder.
func collectTestDefinitions(distDir string) []string {
	testsDir := filepath.Join(distDir, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, filepath.Join(testsDir, e.Name()))
		}
	}
	return paths
}

// generateSBOM generates an SPDX SBOM from a built image tar and writes it alongside the tar.
func generateSBOM(ctx context.Context, sbomTool *syft.SBOMImageTool, tarFile, imageTag string) {
	log.Printf("Generating SBOM for %s ...", imageTag)
	sbomResult, err := sbomTool.GenerateSBOM(ctx, tarFile)
	if err != nil {
		log.Printf("Warning: SBOM generation failed for %s: %v", imageTag, err)
		return
	}
	serialized, err := sbomTool.SerializeSBOM(sbomResult, "spdx-json")
	if err != nil {
		log.Printf("Warning: SBOM serialization failed for %s: %v", imageTag, err)
		return
	}
	sbomPath := tarFile + ".sbom.spdx.json"
	if err := os.WriteFile(sbomPath, serialized, 0644); err != nil {
		log.Printf("Warning: Failed to write SBOM for %s: %v", imageTag, err)
		return
	}
	log.Printf("SBOM written for %s -> %s (%d bytes)", imageTag, sbomPath, len(serialized))
}

// runContainerStructureTests runs container structure tests for a built image tar.
func runContainerStructureTests(dockerClient *docker.Client, tarFile string, testDefs []string, imageTag, reportDir string) {
	if len(testDefs) == 0 {
		log.Printf("No container-structure-test definitions for %s, skipping", imageTag)
		return
	}

	reportFile := filepath.Join(reportDir, fmt.Sprintf("%s-cst-report.xml", strings.ReplaceAll(imageTag, ":", "-")))
	log.Printf("Running container-structure-tests for %s (%d test file(s))...", imageTag, len(testDefs))

	runner := &container_structure_test.TestRunner{
		TestDefinitionPaths: testDefs,
		Image:               tarFile,
		Platform:            platform,
		ReportFile:          reportFile,
		DockerClient:        dockerClient,
	}

	if err := runner.Run(); err != nil {
		log.Printf("Warning: Container structure tests failed for %s: %v", imageTag, err)
		return
	}
	log.Printf("Container structure tests passed for %s -> %s", imageTag, reportFile)
}

// retagAliases creates semantic version tag aliases in the registry for an image.
// It collects all tags (including variant tags), resolves aliases to the highest
// version, and retags them in the registry.
func retagAliases(registryAddr string, imageDef *model.Image) {
	var allTags []string
	for tagName := range imageDef.Tags {
		allTags = append(allTags, tagName)
		for _, variantDef := range imageDef.Variants {
			allTags = append(allTags, tagName+variantDef.TagSuffix)
		}
	}

	aliases := rendering.ResolveAliases(allTags)
	for alias, tag := range aliases {
		sourceRef := fmt.Sprintf("%s/%s:%s", registryAddr, imageDef.Name, tag)
		targetRef := fmt.Sprintf("%s/%s:%s", registryAddr, imageDef.Name, alias)
		log.Printf("Tagging alias %s:%s -> %s:%s", imageDef.Name, alias, imageDef.Name, tag)
		if err := gcr.Retag(sourceRef, targetRef); err != nil {
			log.Printf("Warning: Failed to retag %s -> %s: %v", sourceRef, targetRef, err)
		}
	}
}

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)

	go func() {
		<-done
		os.Exit(0)
	}()

	ctx := context.TODO()
	project, err := discovery.DiscoverProject(ctx, "example")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Discovered %d image(s) in project %s", len(project.ImagesByIdentifier), project.RootDir)
	for name, images := range project.ImagesByName {
		for _, img := range images {
			log.Printf("  image %q: %d tag(s), %d variant(s)", name, len(img.Tags), len(img.Variants))
		}
	}

	distPath := "example/dist"
	if err := rendering.RenderProject(ctx, project, distPath); err != nil {
		log.Fatal(err)
	}
	log.Println("Rendered project to", distPath)

	// Step: Scan rendered Dockerfiles for __hive__/ dependencies
	log.Println("Scanning rendered project for base image dependencies...")
	scannedGraph, err := dependency.ScanRenderedProject(distPath)
	if err != nil {
		log.Fatalf("Dependency scanning failed: %v", err)
	}

	// Step: Merge auto-detected deps with explicit depends_on from image configs
	graph, err := dependency.BuildDependencyGraph(scannedGraph, project)
	if err != nil {
		log.Fatalf("Dependency graph construction failed: %v", err)
	}

	buildOrder, err := graph.TopologicalSort()
	if err != nil {
		log.Fatalf("Dependency resolution failed: %v", err)
	}
	log.Printf("Build order: %v", buildOrder)

	reportDir := "example/reports"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Initialize BuildKit client
	log.Println("Connecting to BuildKit...")
	bkClient, err := buildkit.NewClient(ctx, buildkitAddr)
	if err != nil {
		log.Fatalf("Failed to connect to BuildKit at %s: %v", buildkitAddr, err)
	}
	defer bkClient.Close()

	version, err := bkClient.Version(ctx)
	if err != nil {
		log.Fatalf("Failed to get BuildKit version: %v", err)
	}
	log.Printf("BuildKit version: %s", version)

	// Initialize SBOM tool
	sbomTool, err := syft.NewSBOMImageTool()
	if err != nil {
		log.Fatalf("Failed to initialize SBOM tool: %v", err)
	}

	// Initialize Docker client for container-structure-tests
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	defer dockerClient.Close()

	// Configure S3 cache (matches hack/docker-compose.yml garage service)
	s3Cache := &cache.S3BuildKitCache{
		EndpointUrl:     s3Endpoint,
		Bucket:          s3Bucket,
		Region:          s3Region,
		AccessKeyId:     s3AccessKey,
		SecretAccessKey: s3SecretKey,
		UsePathStyle:    true,
		CacheKey:        imageName,
	}
	log.Printf("S3 cache configured: endpoint=%s, bucket=%s", s3Endpoint, s3Bucket)

	// Step: Build images according to DAG
	if graph.HasDependencies() {
		reg := registry.NewRegistry()
		if err := reg.Start(ctx); err != nil {
			log.Fatalf("Failed to start registry: %v", err)
		}
		defer reg.Stop(ctx)
		log.Printf("Registry started: local=%v address=%s", reg.IsLocal(), reg.Address())

		// Build images in topological order
		for _, imgName := range buildOrder {
			log.Printf("Building image: %s", imgName)

			// Find the image definition
			var imageDef *model.Image
			for _, img := range project.ImagesByIdentifier {
				if img.Name == imgName {
					imageDef = img
					break
				}
			}
			if imageDef == nil {
				log.Printf("Warning: Image %s not found in project", imgName)
				continue
			}

			// Build all tags for this image
			for tagName := range imageDef.Tags {
				// Find the rendered Dockerfile path - format is distPath/imageName/tagName/Dockerfile
				dockerfilePath := filepath.Join(distPath, imgName, tagName, "Dockerfile")
				if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
					log.Fatalf("Dockerfile not found for %s:%s at %s", imgName, tagName, dockerfilePath)
				}

				// Patch hive from container ref
				patchedPath, cleanup := patchHiveRefs(dockerfilePath, reg.Address())
				defer cleanup()

				// Build the image
				root, _ := filepath.Abs(filepath.Dir(patchedPath))
				imageTag := fmt.Sprintf("%s:%s", imgName, tagName)
				tf := tarFilePath(distPath, imgName, tagName)
				build_args, err := buildconfig_resolver.
					ForTag(imageDef, imageDef.Tags[tagName])
				if err != nil {
					log.Fatalf("Failed to resolve build args for variant %s:%s: %v", imgName, tagName, err)
				}

				err = bkClient.Build(ctx, &buildkit.BuildOpts{
					ImageName: imageTag,
					Platform:  platform,
					TarFile:   tf,
					Cache:     s3Cache,
					BuildContext: &build_context.DockerfileBuildContext{
						Root:       root,
						Dockerfile: "Dockerfile.patched",
					},
					BuildArgs: build_args.ToBuildArgs(),
					Secrets:   build_args.Secrets,
				}, newProgressWriter())
				if err != nil {
					log.Printf("Warning: Build failed for %s: %v", imageTag, err)
					continue
				}
				log.Printf("Built %s -> %s", imageTag, tf)

				generateSBOM(ctx, sbomTool, tf, imageTag)
				testDefs := collectTestDefinitions(filepath.Join(distPath, imgName, tagName))
				runContainerStructureTests(dockerClient, tf, testDefs, imageTag, reportDir)

				// Build all variants for this tag
				for variantName, variantDef := range imageDef.Variants {
					variantDockerfilePath := filepath.Join(distPath, imgName, tagName+variantDef.TagSuffix, "Dockerfile")
					if _, err := os.Stat(variantDockerfilePath); os.IsNotExist(err) {
						log.Printf("Warning: Dockerfile not found for variant %s:%s:%s at %s", imgName, tagName, variantName, variantDockerfilePath)
						continue
					}

					variantPatchedPath, variantCleanup := patchHiveRefs(variantDockerfilePath, reg.Address())
					defer variantCleanup()

					variantRoot, _ := filepath.Abs(filepath.Dir(variantPatchedPath))
					variantTag := fmt.Sprintf("%s:%s%s", imgName, tagName, variantDef.TagSuffix)
					variantTf := tarFilePath(distPath, imgName, tagName+variantDef.TagSuffix)

					build_args, err := buildconfig_resolver.
						ForTagVariant(imageDef, variantDef, imageDef.Tags[tagName])
					if err != nil {
						log.Fatalf("Failed to resolve build args for variant %s:%s:%s: %v", imgName, tagName, variantName, err)
					}

					err = bkClient.Build(ctx, &buildkit.BuildOpts{
						ImageName: variantTag,
						Platform:  platform,
						TarFile:   variantTf,
						Cache:     s3Cache,
						BuildContext: &build_context.DockerfileBuildContext{
							Root:       variantRoot,
							Dockerfile: "Dockerfile.patched",
						},
						BuildArgs: build_args.ToBuildArgs(),
					}, newProgressWriter())
					if err != nil {
						log.Printf("Warning: Build failed for variant %s: %v", variantTag, err)
						continue
					}
					log.Printf("Built variant %s -> %s", variantTag, variantTf)

					generateSBOM(ctx, sbomTool, variantTf, variantTag)
					variantTestDefs := collectTestDefinitions(filepath.Join(distPath, imgName, tagName+variantDef.TagSuffix))
					runContainerStructureTests(dockerClient, variantTf, variantTestDefs, variantTag, reportDir)

					// Push variant to local registry if other images depend on it
					if deps := graph.Dependents(imgName); len(deps) > 0 {
						if err := reg.Push(ctx, imgName, tagName+variantDef.TagSuffix, variantTf); err != nil {
							log.Printf("Warning: Failed to push variant %s to registry: %v", variantTag, err)
						} else {
							log.Printf("Pushed variant %s to local registry", variantTag)
						}
					}
				}

				// Push to local registry if other images depend on it
				if deps := graph.Dependents(imgName); len(deps) > 0 {
					if err := reg.Push(ctx, imgName, tagName, tf); err != nil {
						log.Printf("Warning: Failed to push %s:%s to registry: %v", imgName, tagName, err)
					} else {
						log.Printf("Pushed %s:%s to local registry", imgName, tagName)
					}
				}
			}

			// Create semantic version tag aliases in the registry
			retagAliases(reg.Address(), imageDef)
		}
	} else {
		log.Println("No inter-image dependencies, building without registry")

		// Build images in any order (no dependencies)
		for _, images := range project.ImagesByName {
			for _, imageDef := range images {
				log.Printf("Building image: %s", imageDef.Name)

				// Build all tags for this image
				for tagName := range imageDef.Tags {
					// Find the rendered Dockerfile path - format is distPath/imageName/tagName/Dockerfile
					dockerfilePath := filepath.Join(distPath, imageDef.Name, tagName, "Dockerfile")
					if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
						log.Printf("Warning: Dockerfile not found for %s:%s at %s", imageDef.Name, tagName, dockerfilePath)
						continue
					}

					// Build the image
					imageTag := fmt.Sprintf("%s:%s", imageDef.Name, tagName)
					tf := tarFilePath(distPath, imageDef.Name, tagName)

					err = bkClient.Build(ctx, &buildkit.BuildOpts{
						ImageName: imageTag,
						Platform:  platform,
						TarFile:   tf,
						Cache:     s3Cache,
						BuildContext: &build_context.DockerfileBuildContext{
							Root: filepath.Dir(dockerfilePath),
						},
					}, newProgressWriter())
					if err != nil {
						log.Fatalf("Build failed for %s: %v", imageTag, err)
					}
					log.Printf("Built %s -> %s", imageTag, tf)

					generateSBOM(ctx, sbomTool, tf, imageTag)
					testDefs := collectTestDefinitions(filepath.Join(distPath, imageDef.Name, tagName))
					runContainerStructureTests(dockerClient, tf, testDefs, imageTag, reportDir)
				}
			}
		}
	}

}
