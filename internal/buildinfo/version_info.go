package buildinfo

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"
	"time"
)

func addLine(w *tabwriter.Writer, heading string, val string) {
	_, _ = fmt.Fprintf(w, heading+"\t%s\n", val)
}

// PrintVersionInfo writes build metadata to stderr in a tabular format.
func PrintVersionInfo() {
	println()
	println("Build information")
	w := tabwriter.NewWriter(os.Stderr, 10, 1, 10, byte(' '), tabwriter.TabIndent)
	addLine(w, "GitSha", GitSha)
	addLine(w, "Version", Version)
	addLine(w, "BuildTime", BuildTimeParsed.Format(time.DateTime))
	addLine(w, "Go-Version", runtime.Version())
	addLine(w, "OS/Arch", runtime.GOOS+"/"+runtime.GOARCH)
	_ = w.Flush()
}
