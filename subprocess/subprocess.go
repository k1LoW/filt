package subprocess

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/k1LoW/exec"
)

type Subprocess struct {
	ctx     context.Context
	cancel  context.CancelFunc
	command string
	in      io.Reader
	out     io.Writer
}

// NewSubprocess ...
func NewSubprocess(ctx context.Context, command string) *Subprocess {
	innerCtx, cancel := context.WithCancel(ctx)
	return &Subprocess{
		ctx:     innerCtx,
		cancel:  cancel,
		command: command,
	}
}

// Run ...
func (p *Subprocess) Run(in io.Reader) (io.Reader, error) {
	if p.command == "" {
		return in, nil
	}
	r, w := io.Pipe()
	// #nosec
	cmd := exec.CommandContext(p.ctx, "bash", "-c", tuneCommand(p.command))
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Stdin = in
	err := cmd.Start()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		err = w.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		p.cancel()
		return r, err
	}
	go func() {
		err = cmd.Wait()
		if err != nil && err.Error() != "signal: killed" {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		err := w.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		p.cancel()
	}()
	return r, nil
}

func (p *Subprocess) Kill() {
	if p == nil {
		return
	}
	p.cancel()
}

func tuneCommand(command string) string {
	return strings.Replace(command, "grep ", "grep --line-buffered ", -1)
}
