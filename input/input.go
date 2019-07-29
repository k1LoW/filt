package input

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type Input struct{}

// NewInput returns Input
func NewInput() *Input {
	return &Input{}
}

func (i *Input) Handle(ctx context.Context, cancel context.CancelFunc, inn io.Reader) io.Reader {
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
			case <-ctx.Done():
				break L
			default:
				_, err = w.Write(b)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
					break L
				}
			}
		}
		cancel()
	}()

	return r
}
