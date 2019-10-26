package filter

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/k1LoW/filt/history"
	"github.com/k1LoW/filt/input"
	"github.com/k1LoW/filt/output"
	"github.com/k1LoW/filt/subprocess"
	"github.com/nsf/termbox-go"
	"github.com/spf13/viper"
)

func StreamFilter(stdin io.Reader, stdout io.Writer) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i := input.NewInput()

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

	in = i.Handle(ctx, cancel, stdin)

LL:
	for {
		err := termbox.Init()
		if err != nil {
			return exitStatusError, err
		}

		o = output.NewOutput(ctx)
		err = o.Handle(in, stdout)
		if err != nil {
			return exitStatusError, err
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
					o = output.NewOutput(ctx)
					err := o.Handle(in, ioutil.Discard)
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
					sOut, err := s.Run(in)
					if err != nil {
						return exitStatusError, err
					}
					err = h.Append(inputStr)
					if err != nil {
						return exitStatusError, err
					}

					o.Stop()
					in = sOut
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
