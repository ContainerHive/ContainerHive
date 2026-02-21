package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"github.com/timo-reymann/ContainerHive/pkg/cst"
	"github.com/timo-reymann/ContainerHive/pkg/deps"
	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/registry"
	"github.com/timo-reymann/ContainerHive/pkg/rendering"
	"github.com/timo-reymann/ContainerHive/pkg/sbom"
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
	if err := rendering.RenderProject(ctx, project, distPath, nil); err != nil {
		log.Fatal(err)
	}
	log.Println("Rendered project to", distPath)

	// Step: Resolve dependency build order
	log.Println("Scanning rendered project for base image dependencies...")
	buildOrder, err := deps.ResolveOrder(distPath, project)
	if err != nil {
		log.Fatalf("Dependency resolution failed: %v", err)
	}
	log.Printf("Build order: %v", buildOrder.Order())

	// Initialize BuildKit client
	log.Println("Connecting to BuildKit...")
	bkClient, err := build.NewClient(ctx, buildkitAddr)
	if err != nil {
		log.Fatalf("Failed to connect to BuildKit at %s: %v", buildkitAddr, err)
	}
	defer bkClient.Close()

	version, err := bkClient.Version(ctx)
	if err != nil {
		log.Fatalf("Failed to get BuildKit version: %v", err)
	}
	log.Printf("BuildKit version: %s", version)

	// Initialize SBOM generator
	sbomGen, err := sbom.NewGenerator()
	if err != nil {
		log.Fatalf("Failed to initialize SBOM generator: %v", err)
	}

	// Initialize container structure test runner
	cstRunner, err := cst.NewRunner(platform)
	if err != nil {
		log.Fatalf("Failed to initialize CST runner: %v", err)
	}
	defer cstRunner.Close()

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

	// OnBuild callback: generate SBOM and run container structure tests after each build
	onBuild := func(imageTag, tarFile string) {
		// SBOM
		log.Printf("Generating SBOM for %s ...", imageTag)
		sbomData, err := sbomGen.Generate(ctx, tarFile, "spdx-json")
		if err != nil {
			log.Printf("Warning: SBOM generation failed for %s: %v", imageTag, err)
		} else {
			sbomPath := filepath.Join(filepath.Dir(tarFile), "sbom.spdx.json")
			if err := os.WriteFile(sbomPath, sbomData, 0644); err != nil {
				log.Printf("Warning: Failed to write SBOM for %s: %v", imageTag, err)
			} else {
				log.Printf("SBOM written for %s -> %s (%d bytes)", imageTag, sbomPath, len(sbomData))
			}
		}

		// Container structure tests
		testDefs := cst.CollectTestDefinitions(filepath.Dir(tarFile))
		if len(testDefs) == 0 {
			log.Printf("No container-structure-test definitions for %s, skipping", imageTag)
			return
		}
		reportFile := cst.ReportFileName(filepath.Dir(tarFile), imageTag)
		log.Printf("Running container-structure-tests for %s (%d test file(s))...", imageTag, len(testDefs))
		if err := cstRunner.RunTests(tarFile, testDefs, reportFile); err != nil {
			log.Printf("Warning: Container structure tests failed for %s: %v", imageTag, err)
			return
		}
		log.Printf("Container structure tests passed for %s -> %s", imageTag, reportFile)
	}

	// Step: Build images according to DAG
	if buildOrder.HasDependencies() {
		reg := registry.NewRegistry()
		if err := reg.Start(ctx); err != nil {
			log.Fatalf("Failed to start registry: %v", err)
		}
		defer reg.Stop(ctx)
		log.Printf("Registry started: local=%v address=%s", reg.IsLocal(), reg.Address())

		err := build.BuildProject(ctx, bkClient, &build.ProjectBuildOpts{
			Project:     project,
			BuildOrder:  buildOrder,
			DistPath:    distPath,
			Platform:    platform,
			Cache:       s3Cache,
			Registry:    reg,
			ProgressOut: os.Stdout,
			OnBuild:     onBuild,
		})
		if err != nil {
			log.Fatalf("Build failed: %v", err)
		}

		if err := reg.RetagAllAliases(project, []string{}); err != nil {
			log.Fatalf("Retagging failed: %v", err)
		}
	} else {
		log.Println("No inter-image dependencies, building without registry")

		err := build.BuildProject(ctx, bkClient, &build.ProjectBuildOpts{
			Project:     project,
			BuildOrder:  buildOrder,
			DistPath:    distPath,
			Platform:    platform,
			Cache:       s3Cache,
			ProgressOut: os.Stdout,
			OnBuild:     onBuild,
		})
		if err != nil {
			log.Fatalf("Build failed: %v", err)
		}
	}
}
