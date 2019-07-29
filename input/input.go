package input

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type Input struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewInput ...
func NewInput(ctx context.Context, cancel context.CancelFunc) *Input {
	return &Input{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (i *Input) Handle(inn io.Reader) io.Reader {
	r, w := io.Pipe()
	in := bufio.NewReader(inn)
	go func() {
	L:
		for {
			b, err := in.ReadBytes('\n')
			if err == io.EOF {
				break L
			} else if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
				break L
			}
			select {
			case <-i.ctx.Done():
				break L
			default:
				_, err = w.Write(b)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
					break L
				}
			}
		}
		i.cancel()
	}()

	return r
}
