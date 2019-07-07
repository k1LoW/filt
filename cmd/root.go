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
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/k1LoW/filt/version"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

const bufferDisplayLine = 200

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "[STDIN] | filt",
	Short: "filt is a interactive/realtime stream filter",
	Long:  `filt is a interactive/realtime stream filter.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// `--version` option
		versionVal, err := cmd.Flags().GetBool("version")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		if versionVal {
			fmt.Println(version.Version)
			os.Exit(0)
		}

		if terminal.IsTerminal(0) {
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

		var w io.Writer
		output := NewOutput(ctx)
		app := tview.NewApplication()

		var (
			inputStr        string
			currentInputStr string
			s               *Subprocess
		)

		outView := tview.NewTextView().
			SetTextColor(tcell.ColorDefault).
			SetRegions(true).
			SetDynamicColors(true)

		inputField := tview.NewInputField().
			SetLabel("STDOUT | ").
			SetLabelColor(tcell.ColorDarkCyan).
			SetFieldTextColor(tcell.ColorWhite).
			SetFieldBackgroundColor(tcell.ColorDarkCyan).
			SetPlaceholderTextColor(tcell.ColorSilver).
			SetPlaceholder("grep -e GET").
			SetChangedFunc(func(text string) {
				inputStr = text
			}).
			SetDoneFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter:
					s.Kill()
					log.Printf("command: %s", inputStr)
					s = NewSubprocess(ctx, tuneCommand(inputStr))
					output.Stop()
					stdout, err := s.Run(os.Stdin)
					if err != nil {
						log.Printf("err: %s", err)
						_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
						inputStr = currentInputStr
						s = NewSubprocess(ctx, tuneCommand(inputStr))
						stdout, err = s.Run(os.Stdin)
						if err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
							os.Exit(1)
						}
					}
					output = NewOutput(ctx)
					output.Handle(stdout, w)
					outView.Clear()
					currentInputStr = inputStr
				case tcell.KeyCtrlC:
					app.Stop()
				}
			})

		grid := tview.NewGrid().
			SetRows(1).
			SetColumns(0).
			SetBorders(false).
			AddItem(inputField, 0, 0, 1, 1, 0, 0, true).
			AddItem(outView, 1, 0, 1, 1, 0, 0, false)

		go func() {
			t1 := time.NewTicker(10 * time.Millisecond)
			t2 := time.NewTicker(500 * time.Millisecond)
		L:
			for {
				select {
				case <-t1.C:
					app.Draw()
				case <-t2.C:
					outView.Lock()
					current := outView.GetText(false)
					line := strings.Count(current, "\n")
					outView.Unlock()
					if line > bufferDisplayLine*2 {
						splitted := strings.SplitAfterN(current, "\n", line-bufferDisplayLine)
						outView.SetText(strings.TrimSuffix(splitted[line-bufferDisplayLine-1], "\n"))
					}
				case <-ctx.Done():
					break L
				}
			}
		}()

		w = tview.ANSIWriter(outView)

		err := output.Handle(os.Stdin, w)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

		if err := app.SetRoot(grid, true).Run(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "print the version")
}

func tuneCommand(command string) string {
	return strings.Replace(command, "grep ", "grep --line-buffered ", -1)
}
