package registry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/timo-reymann/ContainerHive/internal/utils"
	"zotregistry.dev/zot/v2/pkg/api"
	"zotregistry.dev/zot/v2/pkg/api/config"
)

// ZotRegistry is an embedded OCI registry for local development builds.
// It runs zot in-process on a random port.
type ZotRegistry struct {
	ctlr    *api.Controller
	dataDir string
	tempDir bool
	port    int
}

// NewZotRegistry creates a new ZotRegistry instance.
// If dataDir is non-empty, it is used as persistent storage (created if needed).
// Otherwise a temporary directory is used and cleaned up on Stop.
func NewZotRegistry(dataDir string) *ZotRegistry {
	return &ZotRegistry{dataDir: dataDir}
}

func (z *ZotRegistry) Start(ctx context.Context) error {
	if z.dataDir == "" {
		dataDir, err := os.MkdirTemp("", "containerhive-zot-*")
		if err != nil {
			return errors.Join(errors.New("failed to create zot data directory"), err)
		}
		z.dataDir = dataDir
		z.tempDir = true
	} else {
		if err := os.MkdirAll(z.dataDir, 0755); err != nil {
			return errors.Join(errors.New("failed to create zot data directory"), err)
		}
	}

	conf := config.New()
	conf.HTTP.Address = "0.0.0.0"
	conf.HTTP.Port = "5051"
	conf.Storage.RootDirectory = z.dataDir
	conf.Storage.GC = false
	conf.Storage.Dedupe = false
	conf.Log = &config.LogConfig{
		Level:  "error",
		Output: "",
	}

	z.ctlr = api.NewController(conf)

	if err := z.ctlr.Init(); err != nil {
		if z.tempDir {
			os.RemoveAll(z.dataDir)
		}
		return errors.Join(errors.New("failed to initialize zot"), err)
	}

	go func() {
		if err := z.ctlr.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Run returned unexpectedly; nothing to do since Stop will handle cleanup
		}
	}()

	if err := z.waitForReady(ctx); err != nil {
		z.ctlr.Shutdown()
		if z.tempDir {
			os.RemoveAll(z.dataDir)
		}
		return errors.Join(errors.New("zot failed to become ready"), err)
	}

	return nil
}

func (z *ZotRegistry) waitForReady(ctx context.Context) error {
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return errors.New("timeout waiting for zot to start")
		case <-ticker.C:
			port := z.ctlr.GetPort()
			if port <= 0 {
				continue
			}
			url := fmt.Sprintf("http://0.0.0.0:%d/v2/", port)
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				z.port = port
				return nil
			}
		}
	}
}

func (z *ZotRegistry) Stop(_ context.Context) error {
	if z.ctlr != nil {
		z.ctlr.Shutdown()
	}
	if z.tempDir && z.dataDir != "" {
		os.RemoveAll(z.dataDir)
	}
	return nil
}

func (z *ZotRegistry) Address() string {
	return fmt.Sprintf("%s:%d", outboundIP(), z.port)
}

// outboundIP returns the preferred local IP address used for outbound connections.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func (z *ZotRegistry) IsLocal() bool {
	return true
}

func (z *ZotRegistry) Push(_ context.Context, imageName, tag, ociTarPath string) error {
	tmpDir, err := os.MkdirTemp("", "oci-push-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := utils.ExtractTar(ociTarPath, tmpDir); err != nil {
		return errors.Join(errors.New("failed to extract OCI tar for push"), err)
	}

	layoutPath, err := layout.FromPath(tmpDir)
	if err != nil {
		return errors.Join(errors.New("failed to read OCI layout"), err)
	}

	idx, err := layoutPath.ImageIndex()
	if err != nil {
		return err
	}

	idxManifest, err := idx.IndexManifest()
	if err != nil {
		return err
	}

	if len(idxManifest.Manifests) == 0 {
		return errors.New("no manifests in OCI layout")
	}

	img, err := layoutPath.Image(idxManifest.Manifests[0].Digest)
	if err != nil {
		return errors.Join(errors.New("failed to read image from layout"), err)
	}

	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", z.Address(), imageName, tag), name.Insecure)
	if err != nil {
		return errors.Join(errors.New("invalid image reference"), err)
	}

	if err := remote.Write(ref, img); err != nil {
		return errors.Join(errors.New("failed to push image to zot"), err)
	}

	return nil
}
