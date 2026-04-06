package progress

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/moby/buildkit/client"
)

// DisplayMode controls how build output is rendered.
type DisplayMode int

const (
	// AutoMode detects whether the writer is a TTY: interactive if so, linear otherwise.
	AutoMode DisplayMode = iota
	// InteractiveMode forces TTY-style cursor-movement output.
	InteractiveMode
	// LinearMode forces one-line-per-event output, suitable for CI logs.
	LinearMode
)

// Colors holds ANSI escape sequences for each status type.
// Set a field to an empty string to suppress color for that status.
type Colors struct {
	Running string // step currently executing   — default: bright blue
	Done    string // step completed successfully — default: green
	Cached  string // step result from cache      — default: dim green
	Warning string // build warning               — default: yellow
	Error   string // step failed                 — default: red
	Log     string // script / log output         — default: dim (gray)
	Stage   string // bracketed stage prefix     — default: cyan
	Index   string // step index (#1, #2, …)     — default: dim
	Elapsed string // elapsed time (1.2s)        — default: dim
	Reset   string // reset all attributes        — default: "\x1b[0m"
}

// apply wraps text with the given color code and the Reset sequence.
// Returns text unchanged when color is disabled (empty Reset or empty code).
func (c Colors) apply(code, text string) string {
	if code == "" || c.Reset == "" {
		return text
	}
	return code + text + c.Reset
}

// colorName colorizes any leading bracketed prefix (e.g. "[internal]", "[2/3]")
// in a vertex name using the Stage color.
func (c Colors) colorName(name string) string {
	if c.Stage == "" || c.Reset == "" {
		return name
	}
	// Find the closing bracket of the leading stage prefix.
	if len(name) == 0 || name[0] != '[' {
		return name
	}
	end := strings.Index(name, "] ")
	if end < 0 {
		return name
	}
	bracket := name[:end+1]
	rest := name[end+1:]
	return c.Stage + bracket + c.Reset + rest
}

// DefaultColors returns the color scheme aligned with the tint-based slog handler.
func DefaultColors() Colors {
	return Colors{
		Running: "\x1b[34;1m", // bright blue
		Done:    "\x1b[32m",   // green
		Cached:  "\x1b[92m",   // light green
		Warning: "\x1b[33m",   // yellow
		Error:   "\x1b[31m",   // red
		Log:     "\x1b[2m",    // dim (gray)
		Stage:   "\x1b[36m",   // cyan
		Index:   "\x1b[2m",    // dim
		Elapsed: "\x1b[2m",    // dim
		Reset:   "\x1b[0m",
	}
}

// Config holds all display configuration.
type Config struct {
	Mode   DisplayMode
	Writer io.Writer
	Colors Colors
	// NoColor suppresses all ANSI color output when true.
	// Typically set from os.Getenv("NO_COLOR") != "".
	NoColor bool
}

// NewHandler returns a buildkit status handler function compatible with
// internal/buildkit.Client.Build(). It reads the channel until it is closed,
// rendering each status update according to cfg.
func NewHandler(cfg Config) func(chan *client.SolveStatus) error {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.NoColor {
		cfg.Colors = Colors{}
	}

	return func(ch chan *client.SolveStatus) error {
		mode := cfg.Mode
		if mode == AutoMode {
			if isTTY(cfg.Writer) {
				mode = InteractiveMode
			} else {
				mode = LinearMode
			}
		}

		var d display
		switch mode {
		case InteractiveMode:
			d = newInteractiveDisplay(cfg.Writer, cfg.Colors)
		default:
			d = newLinearDisplay(cfg.Writer, cfg.Colors)
		}

		return runDisplay(context.TODO(), ch, d)
	}
}

// runDisplay drives d until ch is closed or ctx is cancelled.
func runDisplay(ctx context.Context, ch chan *client.SolveStatus, d display) error {
	d.init()
	defer d.done()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case ss, ok := <-ch:
			if !ok {
				return nil
			}
			d.update(ss)
		}
	}
}

// isTTY reports whether w is a character device (terminal).
// Uses only stdlib — no external dependency required.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// display is the internal interface implemented by linearDisplay and interactiveDisplay.
type display interface {
	init()
	update(ss *client.SolveStatus)
	done()
}
