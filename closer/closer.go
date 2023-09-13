package closer

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Func func(ctx context.Context) error

type Closer struct {
	mtx   sync.Mutex
	funcs []Func
}

func (cl *Closer) Add(f Func) {
	cl.mtx.Lock()
	defer cl.mtx.Unlock()

	cl.funcs = append(cl.funcs, f)
}

func (cl *Closer) Close(ctx context.Context) error {
	cl.mtx.Lock()
	defer cl.mtx.Unlock()

	var (
		msgs = make([]string, 0, len(cl.funcs))
		done = make(chan struct{}, 1)
	)

	go func() {
		for _, f := range cl.funcs {
			if err := f(ctx); err != nil {
				msgs = append(msgs, fmt.Sprintf("[!] %v", err))
			}
		}

		done <- struct{}{}
	}()

	select {
	case <-done:
		break
	case <-ctx.Done():
		return fmt.Errorf("shutdown canceled: %v", ctx.Err())
	}

	if len(msgs) > 0 {
		return fmt.Errorf("shutdown finished with error(s): \n%s",
			strings.Join(msgs, "\n"))
	}

	return nil
}
