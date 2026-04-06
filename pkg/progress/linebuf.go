package progress

import (
	"bytes"
	"io"

	digest "github.com/opencontainers/go-digest"
)

// lineBuf buffers partial log lines per vertex and flushes complete lines.
// BuildKit sends log data in arbitrary-sized chunks that may not be
// newline-terminated; this ensures each printed line is complete.
type lineBuf struct {
	bufs map[digest.Digest]*bytes.Buffer
}

func newLineBuf() *lineBuf {
	return &lineBuf{bufs: make(map[digest.Digest]*bytes.Buffer)}
}

// write appends data to the vertex's buffer and calls flush for each complete
// line. Returns the number of complete lines flushed.
func (lb *lineBuf) write(d digest.Digest, data []byte, flush func(line []byte)) {
	buf, ok := lb.bufs[d]
	if !ok {
		buf = &bytes.Buffer{}
		lb.bufs[d] = buf
	}
	buf.Write(data)

	for {
		idx := bytes.IndexByte(buf.Bytes(), '\n')
		if idx < 0 {
			break
		}
		line := make([]byte, idx)
		buf.Read(line)
		buf.ReadByte() // consume the '\n'
		flush(line)
	}
}

// flushRemaining flushes any buffered bytes that were not terminated with '\n'.
// Called when the vertex is done to avoid dropping the last partial line.
func (lb *lineBuf) flushRemaining(d digest.Digest, w io.Writer) {
	buf, ok := lb.bufs[d]
	if !ok || buf.Len() == 0 {
		return
	}
	w.Write(buf.Bytes())
	buf.Reset()
}
