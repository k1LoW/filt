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
	"github.com/k1LoW/filt/config"
	"github.com/k1LoW/filt/history"
	"github.com/k1LoW/filt/input"
	"github.com/k1LoW/filt/output"
	"github.com/k1LoW/filt/subprocess"
	"github.com/k1LoW/filt/version"
	"github.com/mattn/go-isatty"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "filt",
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

		h := history.New(viper.GetString("history.path"))
		if viper.GetBool("history.enable") {
			err := h.Load()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(1)
			}
		}
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
					case termbox.KeyEnter, termbox.KeyCtrlC:
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
						err = h.Append(inputStr)
						if err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
							os.Exit(1)
						}

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
	cobra.OnInitialize(config.Load)
	rootCmd.Flags().BoolP("version", "v", false, "print the version")
	rootCmd.SetUsageTemplate(usageTemplate)
}
