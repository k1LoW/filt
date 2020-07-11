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

func StreamFilter(stdin io.Reader, stdout io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i := input.NewInput()
	o := output.NewOutput(ctx)

	in := i.Handle(ctx, cancel, stdin)

	if err := o.Handle(in, stdout); err != nil {
		return err
	}

	h := history.New(viper.GetString("history.path"))
	if viper.GetBool("history.enable") {
		if err := h.UseHistoryFile(); err != nil {
			return err
		}
	}

	var s *subprocess.Subprocess

LL:
	for {
		if termbox.IsInit {
			termbox.Close()
		}
		if err := termbox.Init(); err != nil {
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
					o = output.NewOutput(ctx)
					if err := o.Handle(in, ioutil.Discard); err != nil {
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
					subStdout, err := s.Run(in)
					if err != nil {
						return err
					}
					if err := h.Append(inputStr); err != nil {
						return err
					}

					o.Stop()
					o = output.NewOutput(ctx)
					if err := o.Handle(subStdout, stdout); err != nil {
						return err
					}
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
