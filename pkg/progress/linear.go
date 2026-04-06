package progress

import (
	"fmt"
	"io"
	"strings"

	"github.com/moby/buildkit/client"
)

// linearDisplay renders one line per event with ANSI colors.
// Suitable for CI environments where cursor movement is not supported.
// Log lines are streamed as they arrive; partial chunks are buffered until
// a newline is received so output is never split mid-line.
type linearDisplay struct {
	w              io.Writer
	colors         Colors
	t              *tracker
	lb             *lineBuf
	seenStatus     map[string]bool
	seenStatusDone map[string]bool
}

func newLinearDisplay(w io.Writer, colors Colors) *linearDisplay {
	return &linearDisplay{w: w, colors: colors, t: newTracker(), lb: newLineBuf(), seenStatus: make(map[string]bool), seenStatusDone: make(map[string]bool)}
}

func (d *linearDisplay) init() {}

func (d *linearDisplay) done() {
	// Flush any log bytes that never received a trailing newline.
	for _, dig := range d.t.order {
		d.lb.flushRemaining(dig, d.w)
	}
}

func (d *linearDisplay) update(ss *client.SolveStatus) {
	d.t.apply(ss)

	// Print status lines for each vertex update.
	for _, bv := range ss.Vertexes {
		v := d.t.vertices[bv.Digest]
		if v == nil {
			continue
		}

		idx := d.colors.apply(d.colors.Index, fmt.Sprintf("#%d", v.index))
		switch {
		case bv.Error != "":
			fmt.Fprintf(d.w, "%s %s %s\n",
				d.colors.apply(d.colors.Error, "✗"),
				idx,
				d.colors.apply(d.colors.Error, fmt.Sprintf("ERROR: %s", strings.TrimSpace(bv.Error))),
			)
		case bv.Completed != nil && bv.Cached:
			fmt.Fprintf(d.w, "%s %s %s  %s\n",
				d.colors.apply(d.colors.Cached, "→"),
				idx,
				d.colors.colorName(v.name),
				d.colors.apply(d.colors.Cached, "CACHED"),
			)
		case bv.Completed != nil:
			fmt.Fprintf(d.w, "%s %s %s  %s %s\n",
				d.colors.apply(d.colors.Done, "✓"),
				idx,
				d.colors.colorName(v.name),
				d.colors.apply(d.colors.Done, "DONE"),
				d.colors.apply(d.colors.Elapsed, v.elapsed()),
			)
		case bv.Started != nil:
			fmt.Fprintf(d.w, "%s %s %s\n",
				d.colors.apply(d.colors.Running, "→"),
				idx,
				d.colors.colorName(v.name),
			)
		}
	}

	// Print status lines (sub-steps like "exporting layers").
	for _, s := range ss.Statuses {
		v := d.t.vertices[s.Vertex]
		prefix := logPrefix(v)

		if s.Completed != nil && !d.seenStatusDone[s.ID] {
			d.seenStatusDone[s.ID] = true
			elapsed := ""
			if s.Started != nil {
				elapsed = " " + d.colors.apply(d.colors.Elapsed, fmt.Sprintf("%.1fs", s.Completed.Sub(*s.Started).Seconds()))
			}
			fmt.Fprintf(d.w, "      %s%s%s %s\n",
				d.colors.apply(d.colors.Log, prefix),
				d.colors.apply(d.colors.Log, s.Name),
				elapsed,
				d.colors.apply(d.colors.Done, "done"),
			)
		} else if s.Completed == nil && !d.seenStatus[s.ID] {
			d.seenStatus[s.ID] = true
			fmt.Fprintf(d.w, "      %s\n", d.colors.apply(d.colors.Log, prefix+s.Name))
		}
	}

	// Stream log lines, buffering partial chunks until a newline is received.
	for _, l := range ss.Logs {
		v := d.t.vertices[l.Vertex]
		prefix := logPrefix(v)
		d.lb.write(l.Vertex, l.Data, func(line []byte) {
			fmt.Fprintf(d.w, "      %s\n", d.colors.apply(d.colors.Log, prefix+string(line)))
		})
	}

	// Print warnings.
	for _, w := range ss.Warnings {
		v := d.t.vertices[w.Vertex]
		prefix := ""
		if v != nil {
			prefix = fmt.Sprintf("#%d ", v.index)
		}
		fmt.Fprintf(d.w, "%s %s%s\n",
			d.colors.apply(d.colors.Warning, "⚠"),
			prefix,
			d.colors.apply(d.colors.Warning, string(w.Short)),
		)
	}
}

// logPrefix returns the "#N | " prefix for a vertex, or empty string if nil.
func logPrefix(v *vertexState) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("#%d | ", v.index)
}
