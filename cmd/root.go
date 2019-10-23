/*
Copyright © 2019 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/k1LoW/filt/input"
	"github.com/k1LoW/filt/output"
	"github.com/k1LoW/filt/subprocess"
	"github.com/k1LoW/filt/version"
	"github.com/mattn/go-isatty"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "[COMMAND] | filt",
	Short: "filt is a interactive/realtime stream filter",
	Long:  `filt is a interactive/realtime stream filter.`,
	Args: func(cmd *cobra.Command, args []string) error {
		versionVal, err := cmd.Flags().GetBool("version")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		if versionVal {
			fmt.Println(version.Version)
			os.Exit(0)
		}

		if isatty.IsTerminal(os.Stdin.Fd()) {
			return errors.New("filt need STDIN. Please use pipe")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.SetOutput(ioutil.Discard)
		if env := os.Getenv("DEBUG"); env != "" {
			debug, err := os.Create("filt.debug")
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(1)
			}
			log.SetOutput(debug)
		}

		i := input.NewInput()
		o := output.NewOutput(ctx)

		in := i.Handle(ctx, cancel, os.Stdin)

		err := o.Handle(in, os.Stdout)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

		history := []string{}
		var s *subprocess.Subprocess

		go func() {
			<-ctx.Done()
			if termbox.IsInit {
				termbox.Interrupt()
			}
		}()

	LL:
		for {
			err = termbox.Init()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(1)
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
						o = output.NewOutput(ctx)
						err := o.Handle(in, ioutil.Discard)
						if err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
							os.Exit(1)
						}
						inputStr := prompt.Input(">>> | ", func(d prompt.Document) []prompt.Suggest {
							s := []prompt.Suggest{}
							for _, h := range history {
								s = append(s, prompt.Suggest{Text: h})
							}
							if d.Text == "" {
								s = append(s, prompt.Suggest{Text: "exit", Description: "exit prompt"})
							}
							return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
						},
							prompt.OptionPrefixTextColor(prompt.Cyan),
							prompt.OptionPreviewSuggestionTextColor(prompt.LightGray),
							prompt.OptionHistory(history),
							prompt.OptionAddKeyBind(prompt.KeyBind{
								Key: prompt.ControlC,
								Fn: func(buf *prompt.Buffer) {
									cancel()
									termbox.Close()
									os.Exit(0)
								}}),
						)
						select {
						case <-ctx.Done():
							termbox.Close()
							break LL
						default:
						}
						if inputStr == "exit" {
							termbox.Close()
							break LL
						}
						s = subprocess.NewSubprocess(ctx, inputStr)
						stdout, err := s.Run(in)
						if err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
							os.Exit(1)
						}
						history = unique(append(history, inputStr))

						o.Stop()
						o = output.NewOutput(ctx)
						err = o.Handle(stdout, os.Stdout)
						if err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
							os.Exit(1)
						}
						break L
					}
				case termbox.EventError:
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", ev.Err)
					os.Exit(1)
				case termbox.EventInterrupt:
					termbox.Close()
					break LL
				}
			}
			termbox.Close()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "print the version")
}

func unique(strs []string) []string {
	keys := make(map[string]bool)
	uniqStrs := []string{}
	for _, s := range strs {
		if s == "" {
			continue
		}
		if _, value := keys[s]; !value {
			keys[s] = true
			uniqStrs = append(uniqStrs, s)
		}
	}
	return uniqStrs
}
