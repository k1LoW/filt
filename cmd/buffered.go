package cmd

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

func runBuffered() (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bufferedIn, err := bufferStdin(ctx, os.Stdin)
	if err != nil {
		return exitStatusError, err
	}

	h := history.New(viper.GetString("history.path"))
	if viper.GetBool("history.enable") {
		err := h.UseHistoryFile()
		if err != nil {
			return exitStatusError, err
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
		err = termbox.Init()
		if err != nil {
			return exitStatusError, err
		}

		o = output.NewOutput(ctx)
		err = o.Handle(in, os.Stdout)
		if err != nil {
			return exitStatusError, err
		}

	L:
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEnter:
					_, _ = fmt.Fprintln(os.Stdout, "")
				case termbox.KeyCtrlC:
					o.Stop()
					s.Kill()
					_, err = bufferedIn.Seek(0, io.SeekStart)
					if err != nil {
						return exitStatusError, err
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
								termbox.Close()
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
					stdout, err := s.Run(bufferedIn)
					if err != nil {
						return exitStatusError, err
					}
					err = h.Append(inputStr)
					if err != nil {
						return exitStatusError, err
					}
					in = stdout
					break L
				}
			case termbox.EventError:
				return exitStatusError, ev.Err
			case termbox.EventInterrupt:
				break LL
			}
		}
		termbox.Close()
	}
	return exitStatusSuccess, nil
}

func bufferStdin(ctx context.Context, stdin io.Reader) (*bytes.Reader, error) {
	r := bufio.NewReader(stdin)
	buf := bytes.NewBuffer(nil)
	line := 0

	ctxB, cancelB := context.WithCancel(ctx)
	defer cancelB()
	err := termbox.Init()
	if err != nil {
		return nil, err
	}

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
					termbox.Close()
					os.Exit(130) // 128 + SIGINT
				}
			}
		}
	}()

E:
	for {
		err = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		if err != nil {
			return nil, err
		}
		b, err := r.ReadBytes('\n')
		if err == io.EOF {
			break E
		} else if err != nil {
			termbox.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			break E
		default:
			_, err = buf.Write(b)
			if err != nil {
				termbox.Close()
				return nil, err
			}
			line = line + 1
			setCellString(1, 1, fmt.Sprintf("%d lines (%d bytes) buffered", line, len(buf.Bytes())), termbox.ColorCyan, termbox.ColorDefault)
		}
		termbox.Flush()
	}
	time.Sleep(1 * time.Second)
	termbox.Close()

	return bytes.NewReader(buf.Bytes()), nil
}

func setCellString(x, y int, s string, fg, bg termbox.Attribute) {
	for _, r := range s {
		termbox.SetCell(x, y, r, fg, bg)
		w := runewidth.RuneWidth(r)
		x += w
	}
}
