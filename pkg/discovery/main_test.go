package discovery

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func TestDiscoverProject(t *testing.T) {
	tests := map[string]struct {
		root     string
		expected *model.ContainerHiveProject
		wantErr  bool
	}{
		"simple project": {
			root: "../testdata/simple-project",
			expected: &model.ContainerHiveProject{
				RootDir:        mustAbs(t, "../testdata/simple-project"),
				ConfigFilePath: mustAbs(t, "../testdata/simple-project/hive.yml"),
				Config: model.HiveProjectConfig{
					Platforms: []string{"linux/amd64"},
				},
				ImagesByIdentifier: map[string]*model.Image{
					"dotnet/8": {
						BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/dotnet/8/Dockerfile"),
						ReadmePath:          mustAbs(t, "../testdata/simple-project/images/dotnet/8/README.md"),
						RootDir:             mustAbs(t, "../testdata/simple-project/images/dotnet/8"),
						RootFSDir:           mustAbs(t, "../testdata/simple-project/images/dotnet/8/rootfs"),
						Identifier:          "dotnet/8",
						Name:                "dotnet",
						DefinitionFilePath:  mustAbs(t, "../testdata/simple-project/images/dotnet/8/image.yml"),
						Secrets:             model.Secrets{},
						Variants: map[string]*model.ImageVariant{
							"node": {
								Name:                "node",
								BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/Dockerfile"),
								ReadmePath:          mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/README.md.gotpl"),
								RootDir:             mustAbs(t, "../testdata/simple-project/images/dotnet/8/node"),
								RootFSDir:           mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/rootfs"),
								TestConfigFilePath:  mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/test.yml.gotpl"),
								TagSuffix:           "-node",
								Versions:            model.Versions{"nodejs": "24"},
								Platforms:           []string{"linux/amd64"},
								Report:              model.ReportModel{Icon: "nodejs-original"},
							},
						},
						BuildArgs: model.BuildArgs{"foo": "bar"},
						Platforms: []string{"linux/amd64"},
						Report:    model.ReportModel{Icon: "dotnet-original"},
						Tags: map[string]*model.Tag{
							"8.0.100": {
								Name: "8.0.100",
								Versions: model.Versions{
									"dotnet-sdk-channel": "8.0.1xx",
								},
							},
							"8.0.200": {
								Name: "8.0.200",
								Versions: model.Versions{
									"dotnet-sdk-channel": "8.0.2xx",
								},
							},
							"8.0.300": {
								Name: "8.0.300",
								Versions: model.Versions{
									"dotnet-sdk-channel": "8.0.3xx",
								},
							},
						},
					},
					"python": {
						RootDir:             mustAbs(t, "../testdata/simple-project/images/python"),
						BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/python/Dockerfile"),
						ReadmePath:          mustAbs(t, "../testdata/simple-project/images/python/README.md"),
						RootFSDir:           mustAbs(t, "../testdata/simple-project/images/python/rootfs"),
						Identifier:          "python",
						Name:                "python",
						BuildArgs:           model.BuildArgs{"foo": "bar"},
						TestConfigFilePath:  mustAbs(t, "../testdata/simple-project/images/python/test.yml.gotpl"),
						DefinitionFilePath:  mustAbs(t, "../testdata/simple-project/images/python/image.yml"),
						Versions: model.Versions{
							"poetry": "2.2.1",
							"uv":     "0.8.22",
						},
						Secrets: model.Secrets{
							"api_key": model.Secret{
								SourceType: "env",
								Value:      "API_KEY",
							},
							"token": model.Secret{
								SourceType: "plain",
								Value:      "secret_token_value",
							},
						},
						Variants: map[string]*model.ImageVariant{},
						Tags: map[string]*model.Tag{
							"3.13.7": {
								Name: "3.13.7",
								Versions: model.Versions{
									"python": "3.13.7",
								},
							},
						},
						Platforms: []string{"linux/amd64"},
					},
				},
				ImagesByName: map[string][]*model.Image{
					"dotnet": {
						{
							Identifier:          "dotnet/8",
							Name:                "dotnet",
							BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/dotnet/8/Dockerfile"),
							ReadmePath:          mustAbs(t, "../testdata/simple-project/images/dotnet/8/README.md"),
							RootDir:             mustAbs(t, "../testdata/simple-project/images/dotnet/8"),
							RootFSDir:           mustAbs(t, "../testdata/simple-project/images/dotnet/8/rootfs"),
							DefinitionFilePath:  mustAbs(t, "../testdata/simple-project/images/dotnet/8/image.yml"),
							BuildArgs:           model.BuildArgs{"foo": "bar"},
							Secrets:             model.Secrets{},
							Variants: map[string]*model.ImageVariant{
								"node": {
									Name:                "node",
									BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/Dockerfile"),
									ReadmePath:          mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/README.md.gotpl"),
									RootDir:             mustAbs(t, "../testdata/simple-project/images/dotnet/8/node"),
									RootFSDir:           mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/rootfs"),
									TestConfigFilePath:  mustAbs(t, "../testdata/simple-project/images/dotnet/8/node/test.yml.gotpl"),
									TagSuffix:           "-node",
									Versions:            model.Versions{"nodejs": "24"},
									Platforms:           []string{"linux/amd64"},
									Report:              model.ReportModel{Icon: "nodejs-original"},
								},
							},
							Platforms: []string{"linux/amd64"},
							Report:    model.ReportModel{Icon: "dotnet-original"},
							Tags: map[string]*model.Tag{
								"8.0.100": {
									Name: "8.0.100",
									Versions: model.Versions{
										"dotnet-sdk-channel": "8.0.1xx",
									},
								},
								"8.0.200": {
									Name: "8.0.200",
									Versions: model.Versions{
										"dotnet-sdk-channel": "8.0.2xx",
									},
								},
								"8.0.300": {
									Name: "8.0.300",
									Versions: model.Versions{
										"dotnet-sdk-channel": "8.0.3xx",
									},
								},
							},
						},
					},
					"python": {
						{
							Identifier:          "python",
							Name:                "python",
							BuildEntryPointPath: mustAbs(t, "../testdata/simple-project/images/python/Dockerfile"),
							ReadmePath:          mustAbs(t, "../testdata/simple-project/images/python/README.md"),
							RootDir:             mustAbs(t, "../testdata/simple-project/images/python"),
							RootFSDir:           mustAbs(t, "../testdata/simple-project/images/python/rootfs"),
							TestConfigFilePath:  mustAbs(t, "../testdata/simple-project/images/python/test.yml.gotpl"),
							DefinitionFilePath:  mustAbs(t, "../testdata/simple-project/images/python/image.yml"),
							BuildArgs:           model.BuildArgs{"foo": "bar"},
							Versions: model.Versions{
								"poetry": "2.2.1",
								"uv":     "0.8.22",
							},
							Secrets: model.Secrets{
								"api_key": model.Secret{
									SourceType: "env",
									Value:      "API_KEY",
								},
								"token": model.Secret{
									SourceType: "plain",
									Value:      "secret_token_value",
								},
							},
							Variants: map[string]*model.ImageVariant{},
							Tags: map[string]*model.Tag{
								"3.13.7": {
									Name: "3.13.7",
									Versions: model.Versions{
										"python": "3.13.7",
									},
								},
							},
							Platforms: []string{"linux/amd64"},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := DiscoverProject(t.Context(), tc.root)
			if (err != nil) != tc.wantErr {
				t.Fatalf("DiscoverProject() error = %v, wantErr %v", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("DiscoverProject() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestDiscoverProject_DependencyProject(t *testing.T) {
	project, err := DiscoverProject(t.Context(), "../testdata/dependency-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("discovers both images", func(t *testing.T) {
		if len(project.ImagesByIdentifier) != 2 {
			t.Errorf("expected 2 images, got %d", len(project.ImagesByIdentifier))
		}
	})

	t.Run("python has depends_on", func(t *testing.T) {
		python := project.ImagesByIdentifier["python"]
		if python == nil {
			t.Fatal("python image not found")
		}
		if len(python.DependsOn) != 1 || python.DependsOn[0] != "ubuntu" {
			t.Errorf("expected depends_on=[ubuntu], got %v", python.DependsOn)
		}
	})

	t.Run("ubuntu has no depends_on", func(t *testing.T) {
		ubuntu := project.ImagesByIdentifier["ubuntu"]
		if ubuntu == nil {
			t.Fatal("ubuntu image not found")
		}
		if len(ubuntu.DependsOn) != 0 {
			t.Errorf("expected empty depends_on, got %v", ubuntu.DependsOn)
		}
	})
}
