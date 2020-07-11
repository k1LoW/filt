/*
Copyright © 2020 Ken'ichiro Oyama <k1lowxb@gmail.com>

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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var out string

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "output shell completion code",
	Long: `output shell completion code.
To configure your shell to load completions for each session

# bash
echo '. <(filt completion bash)' > ~/.bashrc

# zsh
filt completion zsh > $fpath[1]/_filt
`,
	ValidArgs: []string{"bash", "zsh"},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg, received %d", len(args))
		}
		if err := cobra.OnlyValidArgs(cmd, args); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			o   *os.File
			err error
		)
		sh := args[0]
		if out == "" {
			o = os.Stdout
		} else {
			o, err = os.Create(out)
			if err != nil {
				printFatalln(cmd, err)
			}
		}

		switch sh {
		case "bash":
			if err := cmd.Root().GenBashCompletion(o); err != nil {
				_ = o.Close()
				printFatalln(cmd, err)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(o); err != nil {
				_ = o.Close()
				printFatalln(cmd, err)
			}
		}
		if err := o.Close(); err != nil {
			printFatalln(cmd, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.Flags().StringVarP(&out, "out", "o", "", "output file path")
}
