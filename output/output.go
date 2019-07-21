package output

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type Output struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewOutput(ctx context.Context) *Output {
	innerCtx, cancel := context.WithCancel(ctx)
	return &Output{
		ctx:    innerCtx,
		cancel: cancel,
	}
}

func (o *Output) Handle(inn io.Reader, out io.Writer) error {
	in := bufio.NewReader(inn)

	go func() {
	L:
		for {
			b, err := in.ReadBytes('\n')
			if err == io.EOF {
				break
			} else if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(1)
			}
			select {
			case <-o.ctx.Done():
				break L
			default:
				_, err = out.Write(b)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
					os.Exit(1)
				}
			}
		}
	}()

	return nil
}

func (o *Output) Stop() {
	o.cancel()
}
