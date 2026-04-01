package ci

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/timo-reymann/ContainerHive/pkg/templating"
)

// Provider represents a CI provider with its template files.
type Provider struct {
	Name       string
	TemplateFS fs.FS
	Entrypoint string
}

var (
	providers   = make(map[string]*Provider)
	providersMu sync.RWMutex
)

// RegisterProvider registers a CI provider.
func RegisterProvider(p *Provider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[p.Name] = p
}

// GetProvider returns a registered provider by name.
func GetProvider(name string) (*Provider, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()

	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown CI provider: %q", name)
	}
	return p, nil
}

// Generate renders CI configuration for the given provider and context.
// If customTemplateDir is non-empty, templates are loaded from that directory
// instead of the provider's embedded templates.
func Generate(providerName string, ctx *CIContext, customTemplateDir string) ([]byte, error) {
	provider, err := GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	templateFS := provider.TemplateFS
	if customTemplateDir != "" {
		templateFS = os.DirFS(customTemplateDir)
	}

	return templating.RenderWithOptions(templateFS, provider.Entrypoint, ctx, ctx.TemplateOptions)
}
