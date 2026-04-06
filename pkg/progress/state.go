package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/moby/buildkit/client"
	digest "github.com/opencontainers/go-digest"
)

type vertexState struct {
	index     int
	name      string
	started   *time.Time
	completed *time.Time
	cached    bool
	err       string
}

// elapsed returns a human-readable duration string for the vertex (e.g. "1.2s").
// Uses completion time if available, otherwise current time.
func (v *vertexState) elapsed() string {
	if v.started == nil {
		return ""
	}
	end := time.Now()
	if v.completed != nil {
		end = *v.completed
	}
	return fmt.Sprintf("%.1fs", end.Sub(*v.started).Seconds())
}

// tracker maintains vertex state in arrival order.
type tracker struct {
	mu       sync.Mutex
	vertices map[digest.Digest]*vertexState
	order    []digest.Digest
	nextIdx  int
}

func newTracker() *tracker {
	return &tracker{
		vertices: make(map[digest.Digest]*vertexState),
	}
}

// apply merges a SolveStatus update into the tracker and returns the set of
// vertices that changed in this update (new or modified).
func (t *tracker) apply(ss *client.SolveStatus) []*vertexState {
	t.mu.Lock()
	defer t.mu.Unlock()

	changed := make([]*vertexState, 0, len(ss.Vertexes))
	for _, v := range ss.Vertexes {
		state, exists := t.vertices[v.Digest]
		if !exists {
			t.nextIdx++
			state = &vertexState{
				index: t.nextIdx,
				name:  v.Name,
			}
			t.vertices[v.Digest] = state
			t.order = append(t.order, v.Digest)
		}
		state.started = v.Started
		state.completed = v.Completed
		state.cached = v.Cached
		if v.Error != "" {
			state.err = v.Error
		}
		changed = append(changed, state)
	}
	return changed
}

// running returns all vertices that have started but not yet completed.
func (t *tracker) running() []*vertexState {
	t.mu.Lock()
	defer t.mu.Unlock()

	var out []*vertexState
	for _, d := range t.order {
		v := t.vertices[d]
		if v.started != nil && v.completed == nil {
			out = append(out, v)
		}
	}
	return out
}
