package progress

import (
	"fmt"
	"io"
	"strings"

	"github.com/moby/buildkit/client"
)

// interactiveDisplay prints each step header once when it starts, streams log
// lines below it, and prints a completion line when it finishes. All output is
// permanent (no ANSI cursor movement / live region).
type interactiveDisplay struct {
	w              io.Writer
	colors         Colors
	t              *tracker
	lb             *lineBuf
	seenStarted    map[int]bool    // vertex indices already printed as "started" lines
	seenDone       map[int]bool    // vertex indices already printed as permanent lines
	seenStatus     map[string]bool // status IDs already printed as "started"
	seenStatusDone map[string]bool // status IDs already printed as "done"
}

func newInteractiveDisplay(w io.Writer, colors Colors) *interactiveDisplay {
	return &interactiveDisplay{
		w:              w,
		colors:         colors,
		t:              newTracker(),
		lb:             newLineBuf(),
		seenStarted:    make(map[int]bool),
		seenDone:       make(map[int]bool),
		seenStatus:     make(map[string]bool),
		seenStatusDone: make(map[string]bool),
	}
}

func (d *interactiveDisplay) init() {}

func (d *interactiveDisplay) done() {
	// Flush any buffered partial log bytes.
	for _, dig := range d.t.order {
		d.lb.flushRemaining(dig, d.w)
	}
	// Print final state for any vertices still shown as running.
	for _, v := range d.t.running() {
		if !d.seenDone[v.index] {
			d.printDoneLine(v, false, "")
			d.seenDone[v.index] = true
		}
	}
}

func (d *interactiveDisplay) update(ss *client.SolveStatus) {
	d.t.apply(ss)

	// Collect newly started vertices (not yet printed).
	type startedEntry struct {
		v *vertexState
	}
	var started []startedEntry
	for _, bv := range ss.Vertexes {
		v := d.t.vertices[bv.Digest]
		if v == nil || d.seenStarted[v.index] {
			continue
		}
		if bv.Started != nil {
			started = append(started, startedEntry{v})
			d.seenStarted[v.index] = true
		}
	}

	// Collect vertices that completed in this update.
	type completedEntry struct {
		v      *vertexState
		cached bool
		err    string
	}
	var completed []completedEntry
	for _, bv := range ss.Vertexes {
		v := d.t.vertices[bv.Digest]
		if v == nil || d.seenDone[v.index] {
			continue
		}
		if bv.Completed != nil {
			// Ensure a started header exists before printing the done line.
			if !d.seenStarted[v.index] {
				started = append(started, startedEntry{v})
				d.seenStarted[v.index] = true
			}
			completed = append(completed, completedEntry{v, bv.Cached, bv.Error})
			d.seenDone[v.index] = true
		}
	}

	// Collect status lines (sub-steps like "exporting layers", "rewriting layers...").
	// Print each status once when first seen (started) and once when completed.
	type statusLine struct{ prefix, text string }
	var statusLines []statusLine
	for _, s := range ss.Statuses {
		v := d.t.vertices[s.Vertex]
		prefix := logPrefix(v)

		if s.Completed != nil && !d.seenStatusDone[s.ID] {
			d.seenStatusDone[s.ID] = true
			elapsed := ""
			if s.Started != nil {
				elapsed = " " + d.colors.apply(d.colors.Elapsed, fmt.Sprintf("%.1fs", s.Completed.Sub(*s.Started).Seconds()))
			}
			statusLines = append(statusLines, statusLine{prefix, fmt.Sprintf("%s%s %s", s.Name, elapsed, d.colors.apply(d.colors.Done, "done"))})
		} else if s.Completed == nil && !d.seenStatus[s.ID] {
			d.seenStatus[s.ID] = true
			statusLines = append(statusLines, statusLine{prefix, s.Name})
		}
	}

	// Collect log lines for this update (buffered to avoid splitting mid-line).
	type logLine struct{ prefix, text string }
	var logLines []logLine
	for _, l := range ss.Logs {
		v := d.t.vertices[l.Vertex]
		prefix := logPrefix(v)
		d.lb.write(l.Vertex, l.Data, func(line []byte) {
			logLines = append(logLines, logLine{prefix, string(line)})
		})
	}

	// Collect warnings.
	type warnLine struct{ prefix, text string }
	var warnLines []warnLine
	for _, w := range ss.Warnings {
		v := d.t.vertices[w.Vertex]
		prefix := ""
		if v != nil {
			prefix = fmt.Sprintf("#%d ", v.index)
		}
		warnLines = append(warnLines, warnLine{prefix, string(w.Short)})
	}

	// Nothing to redraw.
	if len(started) == 0 && len(completed) == 0 && len(statusLines) == 0 && len(logLines) == 0 && len(warnLines) == 0 {
		return
	}

	// Print newly started vertices (permanent header for the step).
	for _, s := range started {
		fmt.Fprintf(d.w, "  %s %s %s\n",
			d.colors.apply(d.colors.Running, "→"),
			d.colors.apply(d.colors.Index, fmt.Sprintf("#%d", s.v.index)),
			d.colors.colorName(s.v.name),
		)
	}

	// Print status lines (sub-steps like "exporting layers").
	for _, s := range statusLines {
		fmt.Fprintf(d.w, "      %s\n", d.colors.apply(d.colors.Log, s.prefix)+d.colors.apply(d.colors.Log, s.text))
	}

	// Print log lines (permanent, indented below the step header).
	for _, l := range logLines {
		fmt.Fprintf(d.w, "      %s\n", d.colors.apply(d.colors.Log, l.prefix+l.text))
	}

	// Print warnings (permanent).
	for _, w := range warnLines {
		fmt.Fprintf(d.w, "  %s %s%s\n",
			d.colors.apply(d.colors.Warning, "⚠"),
			w.prefix,
			d.colors.apply(d.colors.Warning, w.text),
		)
	}

	// Print newly completed vertices (permanent).
	for _, c := range completed {
		d.printDoneLine(c.v, c.cached, c.err)
	}
}

// printDoneLine writes a single completed-vertex line to the permanent output.
func (d *interactiveDisplay) printDoneLine(v *vertexState, cached bool, errMsg string) {
	idx := d.colors.apply(d.colors.Index, fmt.Sprintf("#%d", v.index))
	switch {
	case errMsg != "":
		fmt.Fprintf(d.w, "  %s %s %s\n",
			d.colors.apply(d.colors.Error, "✗"),
			idx,
			d.colors.apply(d.colors.Error, fmt.Sprintf("%s  ERROR: %s", v.name, strings.TrimSpace(errMsg))),
		)
	case cached:
		fmt.Fprintf(d.w, "  %s %s %s\n",
			d.colors.apply(d.colors.Cached, "→"),
			idx,
			d.colors.apply(d.colors.Cached, "CACHED"),
		)
	default:
		fmt.Fprintf(d.w, "  %s %s %s %s\n",
			d.colors.apply(d.colors.Done, "✓"),
			idx,
			d.colors.apply(d.colors.Done, "DONE"),
			d.colors.apply(d.colors.Elapsed, v.elapsed()),
		)
	}
}
