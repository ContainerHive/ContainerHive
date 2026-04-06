package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/moby/buildkit/client"
	digest "github.com/opencontainers/go-digest"
)

func makeDigest(s string) digest.Digest {
	return digest.Digest("sha256:" + strings.Repeat(s, 64)[:64])
}

func timePtr(t time.Time) *time.Time { return &t }

func sendAndClose(ss ...*client.SolveStatus) chan *client.SolveStatus {
	ch := make(chan *client.SolveStatus, len(ss))
	for _, s := range ss {
		ch <- s
	}
	close(ch)
	return ch
}

func TestLinearDisplay_Running(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Mode:    LinearMode,
		Writer:  &buf,
		Colors:  DefaultColors(),
		NoColor: false,
	}

	now := time.Now()
	d1 := makeDigest("a")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN echo hello", Started: timePtr(now)},
		},
	})

	handler := NewHandler(cfg)
	if err := handler(ch); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "#1") {
		t.Errorf("expected vertex index #1 in output, got: %q", out)
	}
	if !strings.Contains(out, "RUN echo hello") {
		t.Errorf("expected vertex name in output, got: %q", out)
	}
	// Running step should use Running color (bright blue).
	if !strings.Contains(out, DefaultColors().Running) {
		t.Errorf("expected running color in output, got: %q", out)
	}
}

func TestLinearDisplay_Done(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	done := now.Add(2 * time.Second)
	d1 := makeDigest("b")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN make", Started: timePtr(now), Completed: timePtr(done)},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "DONE") {
		t.Errorf("expected DONE in output, got: %q", out)
	}
	if !strings.Contains(out, DefaultColors().Done) {
		t.Errorf("expected done color in output, got: %q", out)
	}
}

func TestLinearDisplay_Cached(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	done := now.Add(100 * time.Millisecond)
	d1 := makeDigest("c")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "FROM ubuntu", Started: timePtr(now), Completed: timePtr(done), Cached: true},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "CACHED") {
		t.Errorf("expected CACHED in output, got: %q", out)
	}
	if !strings.Contains(out, DefaultColors().Cached) {
		t.Errorf("expected cached color in output, got: %q", out)
	}
}

func TestLinearDisplay_Error(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	done := now.Add(time.Second)
	d1 := makeDigest("d")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN false", Started: timePtr(now), Completed: timePtr(done), Error: "exit code 1"},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected ERROR in output, got: %q", out)
	}
	if !strings.Contains(out, "exit code 1") {
		t.Errorf("expected error message in output, got: %q", out)
	}
	if !strings.Contains(out, DefaultColors().Error) {
		t.Errorf("expected error color in output, got: %q", out)
	}
}

func TestLinearDisplay_NoColor(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors(), NoColor: true}

	now := time.Now()
	d1 := makeDigest("e")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN echo", Started: timePtr(now)},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("expected no ANSI codes when NoColor=true, got: %q", out)
	}
}

func TestLinearDisplay_LogLines(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	d1 := makeDigest("f")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN apt-get", Started: timePtr(now)},
		},
		Logs: []*client.VertexLog{
			{Vertex: d1, Data: []byte("Reading package lists...\nDone\n")},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Reading package lists...") {
		t.Errorf("expected log line in output, got: %q", out)
	}
	if !strings.Contains(out, "Done") {
		t.Errorf("expected second log line in output, got: %q", out)
	}
}

func TestLinearDisplay_PartialLogLines(t *testing.T) {
	// BuildKit can send partial chunks without trailing newlines.
	// The line buffer must not split them mid-line.
	var buf bytes.Buffer
	cfg := Config{Mode: LinearMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	d1 := makeDigest("h")
	ch := sendAndClose(
		&client.SolveStatus{
			Vertexes: []*client.Vertex{
				{Digest: d1, Name: "RUN echo", Started: timePtr(now)},
			},
			Logs: []*client.VertexLog{
				{Vertex: d1, Data: []byte("partial")}, // no newline yet
			},
		},
		&client.SolveStatus{
			Logs: []*client.VertexLog{
				{Vertex: d1, Data: []byte(" line\n")}, // completes the line
			},
		},
	)

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "partial line") {
		t.Errorf("expected reassembled partial line in output, got: %q", out)
	}
	// Should appear as a single line, not split across two lines.
	if strings.Contains(out, "partial\n") {
		t.Errorf("partial chunk was printed before its line was complete, got: %q", out)
	}
}

func TestAutoMode_NonTTY(t *testing.T) {
	// bytes.Buffer is not a *os.File, so isTTY returns false → LinearMode used.
	var buf bytes.Buffer
	cfg := Config{Mode: AutoMode, Writer: &buf, Colors: DefaultColors()}

	now := time.Now()
	d1 := makeDigest("g")
	ch := sendAndClose(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{Digest: d1, Name: "RUN test", Started: timePtr(now)},
		},
	})

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce output (linear mode).
	if buf.Len() == 0 {
		t.Error("expected output from AutoMode on non-TTY, got nothing")
	}
}

func TestInteractiveDisplay_StepThenLogs(t *testing.T) {
	// The interactive display should print the step header first,
	// then stream log lines below it.
	var buf bytes.Buffer
	cfg := Config{Mode: InteractiveMode, Writer: &buf, Colors: Colors{}, NoColor: true}

	now := time.Now()
	d1 := makeDigest("i")
	ch := sendAndClose(
		&client.SolveStatus{
			Vertexes: []*client.Vertex{
				{Digest: d1, Name: "RUN apt-get install", Started: timePtr(now)},
			},
			Logs: []*client.VertexLog{
				{Vertex: d1, Data: []byte("Reading package lists...\n")},
			},
		},
		&client.SolveStatus{
			Logs: []*client.VertexLog{
				{Vertex: d1, Data: []byte("Building dependency tree...\n")},
			},
		},
		&client.SolveStatus{
			Vertexes: []*client.Vertex{
				{Digest: d1, Name: "RUN apt-get install", Started: timePtr(now), Completed: timePtr(now.Add(2 * time.Second))},
			},
		},
	)

	if err := NewHandler(cfg)(ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	// Step header must appear before log lines.
	headerIdx := strings.Index(out, "RUN apt-get install")
	logIdx := strings.Index(out, "Reading package lists...")
	if headerIdx < 0 {
		t.Fatalf("step header not found in output: %q", out)
	}
	if logIdx < 0 {
		t.Fatalf("log line not found in output: %q", out)
	}
	if headerIdx > logIdx {
		t.Errorf("step header appeared after log line; expected header first.\nOutput: %q", out)
	}

	if !strings.Contains(out, "Building dependency tree...") {
		t.Errorf("expected second log line in output, got: %q", out)
	}
	if !strings.Contains(out, "DONE") {
		t.Errorf("expected DONE line in output, got: %q", out)
	}
}

func TestIsTTY_Buffer(t *testing.T) {
	var buf bytes.Buffer
	if isTTY(&buf) {
		t.Error("bytes.Buffer should not be a TTY")
	}
}
