package filter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/k1LoW/filt/history"
	"github.com/k1LoW/filt/output"
	"github.com/k1LoW/filt/subprocess"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"github.com/spf13/viper"
)

func BufferFilter(stdin io.Reader, stdout io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bufferedIn, err := bufferStdin(ctx, stdin)
	if err != nil {
		return err
	}

	h := history.New(viper.GetString("history.path"))
	if viper.GetBool("history.enable") {
		if err := h.UseHistoryFile(); err != nil {
			return err
		}
	}

	var (
		in io.Reader
		o  *output.Output
		s  *subprocess.Subprocess
	)

	in = bufferedIn // init in
LL:
	for {
		if termbox.IsInit {
			termbox.Close()
		}
		if err := termbox.Init(); err != nil {
			return err
		}

		o = output.NewOutput(ctx)
		if err := o.Handle(in, stdout); err != nil {
			return err
		}

	L:
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEnter:
					_, _ = fmt.Fprintln(stdout, "")
				case termbox.KeyCtrlC:
					o.Stop()
					s.Kill()
					if _, err := bufferedIn.Seek(0, io.SeekStart); err != nil {
						return err
					}
					inputStr := prompt.Input(">>> | ", func(d prompt.Document) []prompt.Suggest {
						s := []prompt.Suggest{}
						for _, h := range h.Raw() {
							s = append(s, prompt.Suggest{Text: h})
						}
						if d.Text == "" {
							s = append(s, prompt.Suggest{Text: "exit", Description: "exit prompt"})
						}
						return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
					},
						prompt.OptionPrefixTextColor(prompt.Cyan),
						prompt.OptionPreviewSuggestionTextColor(prompt.LightGray),
						prompt.OptionHistory(h.Raw()),
						prompt.OptionAddKeyBind(prompt.KeyBind{
							Key: prompt.ControlC,
							Fn: func(buf *prompt.Buffer) {
								cancel()
								os.Exit(130) // 128 + SIGINT // FIXME: I want not to use os.Exit() in this scope.
							}}),
					)
					select {
					case <-ctx.Done():
						break LL
					default:
					}
					if inputStr == "exit" {
						break LL
					}
					s = subprocess.NewSubprocess(ctx, inputStr)
					sOut, err := s.Run(bufferedIn)
					if err != nil {
						return err
					}
					if err := h.Append(inputStr); err != nil {
						return err
					}
					in = sOut
					break L
				}
			case termbox.EventError:
				return ev.Err
			case termbox.EventInterrupt:
				break LL
			}
		}
	}
	return nil
}

func bufferStdin(ctx context.Context, stdin io.Reader) (*bytes.Reader, error) {
	r := bufio.NewReader(stdin)
	buf := bytes.NewBuffer(nil)
	line := 0

	ctxB, cancelB := context.WithCancel(ctx)
	defer cancelB()
	if err := termbox.Init(); err != nil {
		return nil, err
	}
	defer termbox.Close()

	go func() {
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				select {
				case <-ctxB.Done():
					return
				default:
				}
				switch ev.Key {
				case termbox.KeyCtrlC:
					// Cancel buffering
					cancelB()
				}
			}
		}
	}()

E:
	for {
		if err := termbox.Clear(termbox.ColorDefault, termbox.ColorDefault); err != nil {
			return nil, err
		}
		b, err := r.ReadBytes('\n')
		if err == io.EOF {
			break E
		} else if err != nil {
			return nil, err
		}
		select {
		case <-ctxB.Done():
			break E
		default:
			if _, err := buf.Write(b); err != nil {
				return nil, err
			}
			line = line + 1
			setCellString(0, 0, fmt.Sprintf("%d lines (%d bytes) buffered", line, len(buf.Bytes())), termbox.ColorCyan, termbox.ColorDefault)
		}
		if err := termbox.Flush(); err != nil {
			return nil, err
		}
	}

	time.Sleep(1 * time.Second)

	return bytes.NewReader(buf.Bytes()), nil
}

func setCellString(x, y int, s string, fg, bg termbox.Attribute) {
	for _, r := range s {
		termbox.SetCell(x, y, r, fg, bg)
		w := runewidth.RuneWidth(r)
		x += w
	}
}
