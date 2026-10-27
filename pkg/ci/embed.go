package ci

import (
	"embed"
	"io/fs"
)

//go:embed templates/gitlab/*.gotpl
var gitlabTemplates embed.FS

//go:embed templates/github/*.gotpl
var githubTemplates embed.FS

func init() {
	RegisterProvider(&Provider{
		Name:       "gitlab",
		TemplateFS: mustSub(gitlabTemplates, "templates/gitlab"),
		Entrypoint: "pipeline.yml.gotpl",
		// GitLab runners pull anonymously from Docker Hub and hit its rate
		// limits, the GitLab registry is the closest mirror.
		DefaultImageName: GitLabImageName,
	})
	RegisterProvider(&Provider{
		Name:       "github",
		TemplateFS: mustSub(githubTemplates, "templates/github"),
		Entrypoint: "workflow.yml.gotpl",
		// GitHub-hosted runners pull anonymously from Docker Hub and hit its
		// rate limits, ghcr.io has none.
		DefaultImageName: GHCRImageName,
	})
}

func mustSub(fsys embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
