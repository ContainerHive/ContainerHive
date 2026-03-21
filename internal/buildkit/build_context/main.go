package build_context

import (
	"context"

	gatewayClient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/tonistiigi/fsutil"
)

// BuildContext defines the interface for providing a build context to BuildKit.
type BuildContext interface {
	// FrontendType returns the BuildKit frontend identifier (e.g. "dockerfile.v0").
	FrontendType() string
	// ToLocalMounts returns the filesystem mounts required for the build.
	ToLocalMounts() (map[string]fsutil.FS, error)
	// FileName returns the name of the build entry point file.
	FileName() string
	// RunBuild executes the build using the given BuildKit gateway client.
	RunBuild(context.Context, gatewayClient.Client) (*gatewayClient.Result, error)
}
