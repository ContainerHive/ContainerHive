package tagrange

import (
	"fmt"
	"time"

	"github.com/blues/jsonata-go"
)

// evalTimeout bounds a single JSONata evaluation. jsonata-go has no
// built-in cancellation, so a user expression that runs unbounded over a
// large document would otherwise hang discovery indefinitely.
const evalTimeout = 5 * time.Second

// EvalTransform compiles and evaluates a JSONata expression against data,
// wrapped behind this function (rather than exposing jsonata.Expr directly)
// so the engine can be swapped without touching callers. jsonata-go targets
// jsonata-js 1.5.4; an expression copied from a newer JSONata Exerciser may
// not compile here.
func EvalTransform(expr string, data any) (any, error) {
	compiled, err := jsonata.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("transform %q failed to compile (containerhive uses the JSONata 1.5.4 subset): %w", expr, err)
	}

	// *jsonata.Expr's concurrency safety is undocumented, so each call
	// compiles its own expression rather than sharing one across
	// goroutines; resolution runs at most once per range per process
	// (results are cached upstream), so recompiling is cheap.
	type result struct {
		val any
		err error
	}
	done := make(chan result, 1)
	go func() {
		val, err := compiled.Eval(data)
		done <- result{val, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("transform %q failed: %w", expr, r.err)
		}
		return r.val, nil
	case <-time.After(evalTimeout):
		return nil, fmt.Errorf("transform %q exceeded %s - simplify the expression", expr, evalTimeout)
	}
}
