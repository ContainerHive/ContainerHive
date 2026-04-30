package rendering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/discovery"
)

func discoverAndRender(t *testing.T, projectPath string) string {
	t.Helper()
	project, err := discovery.DiscoverProject(t.Context(), projectPath)
	if err != nil {
		t.Fatalf("failed to discover project: %v", err)
	}
	targetPath := filepath.Join(t.TempDir(), "dist")
	if err := RenderProject(t.Context(), project, targetPath); err != nil {
		t.Fatalf("failed to render project: %v", err)
	}
	return targetPath
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
		return
	}
	if stat.IsDir() {
		t.Errorf("expected %s to be a file, got directory", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected directory %s to exist: %v", path, err)
		return
	}
	if !stat.IsDir() {
		t.Errorf("expected %s to be a directory, got file", path)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		t.Errorf("expected %s to not exist, but it does", path)
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read %s: %v", path, err)
		return
	}
	if string(got) != expected {
		t.Errorf("file %s content mismatch:\n  expected: %q\n  got:      %q", path, expected, string(got))
	}
}

func assertFileContains(t *testing.T, path, substring string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read %s: %v", path, err)
		return
	}
	if !strings.Contains(string(got), substring) {
		t.Errorf("file %s does not contain %q, got:\n%s", path, substring, string(got))
	}
}

func TestRenderProject_MinimalProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/minimal-project")

	t.Run("creates image directory", func(t *testing.T) {
		assertDirExists(t, filepath.Join(dist, "nginx"))
	})

	t.Run("creates tag directory with Dockerfile", func(t *testing.T) {
		tagDir := filepath.Join(dist, "nginx", "1.27")
		assertDirExists(t, tagDir)
		assertFileExists(t, filepath.Join(tagDir, "Dockerfile"))
		assertFileContains(t, filepath.Join(tagDir, "Dockerfile"), "FROM nginx:alpine")
	})

	t.Run("does not create tests directory", func(t *testing.T) {
		assertNotExists(t, filepath.Join(dist, "nginx", "1.27", "tests"))
	})

	t.Run("does not create rootfs directory", func(t *testing.T) {
		assertNotExists(t, filepath.Join(dist, "nginx", "1.27", "rootfs"))
	})
}

func TestRenderProject_TemplateProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/template-project")

	tagDir := filepath.Join(dist, "app", "latest")

	t.Run("renders Dockerfile.gotpl with version", func(t *testing.T) {
		// Build entrypoint preserves original filename
		df := filepath.Join(tagDir, "Dockerfile")
		assertFileExists(t, df)
		assertFileContains(t, df, "FROM golang:1.22.5")
	})

	t.Run("renders test config with version and image name", func(t *testing.T) {
		testFile := filepath.Join(tagDir, "tests", "image.yml")
		assertFileExists(t, testFile)
		assertFileContains(t, testFile, "go1.22.5")
		assertFileContains(t, testFile, "\"app\"")
	})

	t.Run("copies rootfs", func(t *testing.T) {
		confFile := filepath.Join(tagDir, "rootfs", "etc", "app.conf")
		assertFileExists(t, confFile)
		assertFileContains(t, confFile, "env=production")
	})
}

func TestRenderProject_SimpleProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/simple-project")

	t.Run("python image", func(t *testing.T) {
		tagDir := filepath.Join(dist, "python", "3.13.7")

		t.Run("creates tag directory", func(t *testing.T) {
			assertDirExists(t, tagDir)
		})

		t.Run("copies Dockerfile", func(t *testing.T) {
			df := filepath.Join(tagDir, "Dockerfile")
			assertFileExists(t, df)
			assertFileContains(t, df, "FROM base")
			assertFileContains(t, df, "pyenv install")
		})

		t.Run("copies rootfs", func(t *testing.T) {
			assertDirExists(t, filepath.Join(tagDir, "rootfs"))
			assertFileExists(t, filepath.Join(tagDir, "rootfs", "etc", "some-config", "value.yaml"))
		})

		t.Run("renders test config with python version", func(t *testing.T) {
			testFile := filepath.Join(tagDir, "tests", "image.yml")
			assertFileExists(t, testFile)
			assertFileContains(t, testFile, "Python 3.13.7")
		})
	})

	t.Run("dotnet image", func(t *testing.T) {
		dotnetDir := filepath.Join(dist, "dotnet")
		assertDirExists(t, dotnetDir)

		t.Run("creates all tag directories", func(t *testing.T) {
			for _, tag := range []string{"8.0.100", "8.0.200", "8.0.300"} {
				assertDirExists(t, filepath.Join(dotnetDir, tag))
			}
		})

		t.Run("tag directory has Dockerfile and rootfs", func(t *testing.T) {
			tagDir := filepath.Join(dotnetDir, "8.0.100")
			assertFileExists(t, filepath.Join(tagDir, "Dockerfile"))
			assertFileContains(t, filepath.Join(tagDir, "Dockerfile"), "install-dotnet")
			assertFileExists(t, filepath.Join(tagDir, "rootfs", "opt", "acme-corp", "info"))
			assertFileContent(t, filepath.Join(tagDir, "rootfs", "opt", "acme-corp", "info"), "source=image")
		})

		t.Run("tag directory has no tests folder", func(t *testing.T) {
			// dotnet/8 has no test config at image level
			assertNotExists(t, filepath.Join(dotnetDir, "8.0.100", "tests"))
		})

		t.Run("creates variant directories with tag suffix", func(t *testing.T) {
			for _, tag := range []string{"8.0.100", "8.0.200", "8.0.300"} {
				assertDirExists(t, filepath.Join(dotnetDir, tag+"-node"))
			}
		})

		t.Run("variant has own Dockerfile", func(t *testing.T) {
			variantDir := filepath.Join(dotnetDir, "8.0.100-node")
			df := filepath.Join(variantDir, "Dockerfile")
			assertFileExists(t, df)
			assertFileContains(t, df, "nodesource")
		})

		t.Run("variant rootfs overlays image rootfs", func(t *testing.T) {
			variantDir := filepath.Join(dotnetDir, "8.0.100-node")
			infoFile := filepath.Join(variantDir, "rootfs", "opt", "acme-corp", "info")
			assertFileExists(t, infoFile)
			// Variant rootfs should overwrite image rootfs file
			assertFileContent(t, infoFile, "source=variant")
		})

		t.Run("variant has test config from variant only", func(t *testing.T) {
			variantDir := filepath.Join(dotnetDir, "8.0.100-node")
			testsDir := filepath.Join(variantDir, "tests")
			assertDirExists(t, testsDir)

			// No image-level test config
			assertNotExists(t, filepath.Join(testsDir, "image.yml"))

			// Variant test config rendered with nodejs version
			variantTest := filepath.Join(testsDir, "variant.yml")
			assertFileExists(t, variantTest)
			assertFileContains(t, variantTest, "24")
		})
	})
}

func TestRenderProject_DependencyProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/dependency-project")

	t.Run("preserves __hive__/ prefix in plain Dockerfile", func(t *testing.T) {
		df := filepath.Join(dist, "python", "3.13", "Dockerfile")
		assertFileExists(t, df)
		assertFileContains(t, df, "FROM __hive__/ubuntu:22.04")
	})

	t.Run("ubuntu has plain FROM", func(t *testing.T) {
		df := filepath.Join(dist, "ubuntu", "22.04", "Dockerfile")
		assertFileExists(t, df)
		assertFileContains(t, df, "FROM ubuntu:22.04")
	})
}

func TestRenderProject_DependencyTemplateProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/dependency-template-project")

	t.Run("renders resolve_base to __hive__/ prefix", func(t *testing.T) {
		df := filepath.Join(dist, "app", "latest", "Dockerfile")
		assertFileExists(t, df)
		assertFileContains(t, df, "FROM __hive__/ubuntu:22.04")
	})
}

func TestResolveAliases(t *testing.T) {
	t.Run("picks highest version per alias", func(t *testing.T) {
		tags := []string{"8.0.100", "8.0.200", "8.0.300"}
		aliases := ResolveAliases(tags)

		if got := aliases["8.0"]; got != "8.0.300" {
			t.Errorf("alias 8.0: expected 8.0.300, got %q", got)
		}
		if got := aliases["8"]; got != "8.0.300" {
			t.Errorf("alias 8: expected 8.0.300, got %q", got)
		}
	})

	t.Run("handles variant suffixes independently", func(t *testing.T) {
		tags := []string{"8.0.100", "8.0.300", "8.0.100-node", "8.0.300-node"}
		aliases := ResolveAliases(tags)

		if got := aliases["8.0"]; got != "8.0.300" {
			t.Errorf("alias 8.0: expected 8.0.300, got %q", got)
		}
		if got := aliases["8.0-node"]; got != "8.0.300-node" {
			t.Errorf("alias 8.0-node: expected 8.0.300-node, got %q", got)
		}
		if got := aliases["8-node"]; got != "8.0.300-node" {
			t.Errorf("alias 8-node: expected 8.0.300-node, got %q", got)
		}
	})

	t.Run("skips non-semver tags", func(t *testing.T) {
		tags := []string{"latest", "1.2.3"}
		aliases := ResolveAliases(tags)

		if got := aliases["1.2"]; got != "1.2.3" {
			t.Errorf("alias 1.2: expected 1.2.3, got %q", got)
		}
		if _, ok := aliases["latest"]; ok {
			t.Error("expected no alias for 'latest'")
		}
	})

	t.Run("single tag with no lower variants", func(t *testing.T) {
		tags := []string{"1"}
		aliases := ResolveAliases(tags)

		if len(aliases) != 0 {
			t.Errorf("expected no aliases for major-only tag, got %v", aliases)
		}
	})
}

func TestResolveLatestAlias(t *testing.T) {
	t.Run("empty alias returns empty string and no error", func(t *testing.T) {
		target, err := ResolveLatestAlias([]string{"1.0.0"}, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if target != "" {
			t.Errorf("expected empty target, got %q", target)
		}
	})

	t.Run("no semantic tags returns error", func(t *testing.T) {
		_, err := ResolveLatestAlias([]string{"foo", "bar"}, "latest")
		if err == nil {
			t.Error("expected error for non-semantic tags, got nil")
		}
	})

	t.Run("single semantic tag points alias to it", func(t *testing.T) {
		target, err := ResolveLatestAlias([]string{"1.0.0"}, "latest")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if target != "1.0.0" {
			t.Errorf("expected 1.0.0, got %q", target)
		}
	})

	t.Run("multiple semantic tags points alias to highest", func(t *testing.T) {
		target, err := ResolveLatestAlias([]string{"8.1.0", "8.2.0", "8.0.100"}, "latest")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if target != "8.2.0" {
			t.Errorf("expected 8.2.0, got %q", target)
		}
	})

	t.Run("mixed semantic and non-semantic tags uses only semantic ones", func(t *testing.T) {
		target, err := ResolveLatestAlias([]string{"foo", "1.0.0", "bar", "2.0.0"}, "stable")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if target != "2.0.0" {
			t.Errorf("expected 2.0.0, got %q", target)
		}
	})
}

// ---------------------------------------------------------------------------
// Unit tests for low-coverage functions
// ---------------------------------------------------------------------------

func TestReplaceHiveParent(t *testing.T) {
	t.Run("replaces __hive_parent__ with concrete reference", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "Dockerfile")
		os.WriteFile(f, []byte("FROM __hive_parent__\nRUN apt-get update"), 0644)

		if err := replaceHiveParent(f, "myimg", "1.0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, f, "FROM __hive__/myimg:1.0\nRUN apt-get update")
	})

	t.Run("file without placeholder is unchanged", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "Dockerfile")
		original := "FROM ubuntu:22.04\nRUN echo hello"
		os.WriteFile(f, []byte(original), 0644)

		if err := replaceHiveParent(f, "myimg", "1.0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, f, original)
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		err := replaceHiveParent(filepath.Join(t.TempDir(), "missing"), "img", "1.0")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("multiple occurrences are all replaced", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "Dockerfile")
		os.WriteFile(f, []byte("FROM __hive_parent__\nCOPY --from=__hive_parent__ /bin /bin"), 0644)

		if err := replaceHiveParent(f, "myimg", "1.0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, f, "FROM __hive__/myimg:1.0\nCOPY --from=__hive__/myimg:1.0 /bin /bin")
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("copies file content correctly", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src.txt")
		dst := filepath.Join(tmp, "dst.txt")
		os.WriteFile(src, []byte("hello world"), 0644)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, dst, "hello world")
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "exec.sh")
		dst := filepath.Join(tmp, "exec_copy.sh")
		os.WriteFile(src, []byte("#!/bin/sh"), 0755)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		srcInfo, _ := os.Stat(src)
		dstInfo, _ := os.Stat(dst)
		if srcInfo.Mode() != dstInfo.Mode() {
			t.Errorf("permission mismatch: src=%v dst=%v", srcInfo.Mode(), dstInfo.Mode())
		}
	})

	t.Run("source does not exist returns error", func(t *testing.T) {
		tmp := t.TempDir()
		err := copyFile(filepath.Join(tmp, "nope"), filepath.Join(tmp, "dst"))
		if err == nil {
			t.Fatal("expected error for missing source")
		}
	})

	t.Run("destination dir does not exist returns error", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src.txt")
		os.WriteFile(src, []byte("data"), 0644)

		err := copyFile(src, filepath.Join(tmp, "no", "such", "dir", "dst.txt"))
		if err == nil {
			t.Fatal("expected error for missing destination directory")
		}
	})
}

func TestCopyDir(t *testing.T) {
	t.Run("copies directory tree recursively", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		os.MkdirAll(filepath.Join(src, "sub"), 0755)
		os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0644)
		os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("bbb"), 0644)

		dst := filepath.Join(tmp, "dst")
		if err := copyDir(src, dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertFileContent(t, filepath.Join(dst, "a.txt"), "aaa")
		assertFileContent(t, filepath.Join(dst, "sub", "b.txt"), "bbb")
	})

	t.Run("preserves file content", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		os.MkdirAll(src, 0755)
		content := "line1\nline2\nline3"
		os.WriteFile(filepath.Join(src, "file.txt"), []byte(content), 0644)

		dst := filepath.Join(tmp, "dst")
		if err := copyDir(src, dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, filepath.Join(dst, "file.txt"), content)
	})

	t.Run("source does not exist returns error", func(t *testing.T) {
		tmp := t.TempDir()
		err := copyDir(filepath.Join(tmp, "missing"), filepath.Join(tmp, "dst"))
		if err == nil {
			t.Fatal("expected error for missing source")
		}
	})

	t.Run("empty directory creates empty destination", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "empty")
		os.MkdirAll(src, 0755)

		dst := filepath.Join(tmp, "dst")
		if err := copyDir(src, dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDirExists(t, dst)

		entries, err := os.ReadDir(dst)
		if err != nil {
			t.Fatalf("failed to read dst dir: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected empty directory, got %d entries", len(entries))
		}
	})
}

func TestCopyRootFs(t *testing.T) {
	t.Run("copies source dir into targetRoot/rootfs", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		os.MkdirAll(filepath.Join(src, "etc"), 0755)
		os.WriteFile(filepath.Join(src, "etc", "app.conf"), []byte("key=val"), 0644)

		target := filepath.Join(tmp, "target")
		if err := copyRootFs(src, target); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDirExists(t, filepath.Join(target, "rootfs"))
		assertFileContent(t, filepath.Join(target, "rootfs", "etc", "app.conf"), "key=val")
	})

	t.Run("creates target root if needed", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		os.MkdirAll(src, 0755)
		os.WriteFile(filepath.Join(src, "f.txt"), []byte("hi"), 0644)

		target := filepath.Join(tmp, "new", "nested", "target")
		if err := copyRootFs(src, target); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDirExists(t, filepath.Join(target, "rootfs"))
		assertFileContent(t, filepath.Join(target, "rootfs", "f.txt"), "hi")
	})

	t.Run("source does not exist returns error", func(t *testing.T) {
		tmp := t.TempDir()
		err := copyRootFs(filepath.Join(tmp, "no-src"), filepath.Join(tmp, "target"))
		if err == nil {
			t.Fatal("expected error for missing source")
		}
	})
}

func TestFixUpEntrypoint(t *testing.T) {
	t.Run("strips template extension", func(t *testing.T) {
		got := fixUpEntrypoint("/out", "/in/Dockerfile.gotpl")
		expected := "/out/Dockerfile"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("plain filename without template extension", func(t *testing.T) {
		got := fixUpEntrypoint("/out", "/in/Dockerfile")
		expected := "/out/Dockerfile"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestCreateTestsFolder(t *testing.T) {
	t.Run("creates tests subdirectory inside root", func(t *testing.T) {
		tmp := t.TempDir()
		got, err := createTestsFolder(tmp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(tmp, "tests")
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
		assertDirExists(t, expected)
	})

	t.Run("returns correct path", func(t *testing.T) {
		tmp := t.TempDir()
		got, err := createTestsFolder(tmp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(got, "/tests") {
			t.Errorf("expected path ending in /tests, got %q", got)
		}
	})

	t.Run("root does not exist still works because mkdir creates it", func(t *testing.T) {
		tmp := t.TempDir()
		root := filepath.Join(tmp, "nonexistent")
		got, err := createTestsFolder(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDirExists(t, got)
	})
}

func TestRenderProject_MultiVariantProject(t *testing.T) {
	dist := discoverAndRender(t, "../testdata/multi-variant-project")

	baseDir := filepath.Join(dist, "base")
	assertDirExists(t, baseDir)

	t.Run("tag directory", func(t *testing.T) {
		tagDir := filepath.Join(baseDir, "3.3.0")

		t.Run("has Dockerfile", func(t *testing.T) {
			assertFileExists(t, filepath.Join(tagDir, "Dockerfile"))
			assertFileContains(t, filepath.Join(tagDir, "Dockerfile"), "FROM ruby:alpine")
		})

		t.Run("has rootfs from image", func(t *testing.T) {
			assertFileExists(t, filepath.Join(tagDir, "rootfs", "etc", "base.conf"))
			assertFileContains(t, filepath.Join(tagDir, "rootfs", "etc", "base.conf"), "source=base")
		})

		t.Run("has rendered test config", func(t *testing.T) {
			testFile := filepath.Join(tagDir, "tests", "image.yml")
			assertFileExists(t, testFile)
			assertFileContains(t, testFile, "ruby 3.3.0")
		})
	})

	t.Run("slim variant", func(t *testing.T) {
		slimDir := filepath.Join(baseDir, "3.3.0-slim")
		assertDirExists(t, slimDir)

		t.Run("has variant Dockerfile", func(t *testing.T) {
			assertFileExists(t, filepath.Join(slimDir, "Dockerfile"))
			assertFileContains(t, filepath.Join(slimDir, "Dockerfile"), "FROM ruby:slim")
		})

		t.Run("has image rootfs", func(t *testing.T) {
			assertFileExists(t, filepath.Join(slimDir, "rootfs", "etc", "base.conf"))
			assertFileContains(t, filepath.Join(slimDir, "rootfs", "etc", "base.conf"), "source=base")
		})

		t.Run("has variant rootfs", func(t *testing.T) {
			assertFileExists(t, filepath.Join(slimDir, "rootfs", "etc", "slim.conf"))
			assertFileContains(t, filepath.Join(slimDir, "rootfs", "etc", "slim.conf"), "variant=slim")
		})

		t.Run("has image test config but no variant test config", func(t *testing.T) {
			assertFileExists(t, filepath.Join(slimDir, "tests", "image.yml"))
			assertFileContains(t, filepath.Join(slimDir, "tests", "image.yml"), "ruby 3.3.0")
			assertNotExists(t, filepath.Join(slimDir, "tests", "variant.yml"))
		})
	})

	t.Run("full variant", func(t *testing.T) {
		fullDir := filepath.Join(baseDir, "3.3.0-full")
		assertDirExists(t, fullDir)

		t.Run("has variant Dockerfile", func(t *testing.T) {
			assertFileExists(t, filepath.Join(fullDir, "Dockerfile"))
			assertFileContains(t, filepath.Join(fullDir, "Dockerfile"), "FROM ruby:latest")
		})

		t.Run("rootfs overlay overwrites image files", func(t *testing.T) {
			baseConf := filepath.Join(fullDir, "rootfs", "etc", "base.conf")
			assertFileExists(t, baseConf)
			// full variant rootfs overwrites the image-level base.conf
			assertFileContains(t, baseConf, "source=full-override")
		})

		t.Run("rootfs overlay adds variant files", func(t *testing.T) {
			assertFileExists(t, filepath.Join(fullDir, "rootfs", "etc", "full.conf"))
			assertFileContains(t, filepath.Join(fullDir, "rootfs", "etc", "full.conf"), "variant=full")
		})

		t.Run("has both image and variant test configs", func(t *testing.T) {
			imageTest := filepath.Join(fullDir, "tests", "image.yml")
			assertFileExists(t, imageTest)
			assertFileContains(t, imageTest, "ruby 3.3.0")

			variantTest := filepath.Join(fullDir, "tests", "variant.yml")
			assertFileExists(t, variantTest)
			assertFileContains(t, variantTest, "enabled")
		})
	})
}
