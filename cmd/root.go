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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/k1LoW/filt/config"
	"github.com/k1LoW/filt/filter"
	"github.com/k1LoW/filt/version"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
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

var buffered bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "filt",
	Short: "filt is a interactive/realtime stream filter",
	Long:  `filt is a interactive/realtime stream filter.`,
	Args: func(cmd *cobra.Command, args []string) error {
		versionVal, err := cmd.Flags().GetBool("version")
		if err != nil {
			printFatalln(cmd, err)
		}
		if versionVal {
			cmd.Println(version.Version)
			os.Exit(0)
		}

		if isatty.IsTerminal(os.Stdin.Fd()) {
			return errors.New("filt need STDIN. Please use pipe")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			err error
		)
		if buffered {
			err = filter.BufferFilter(os.Stdin, os.Stdout)
		} else {
			err = filter.StreamFilter(os.Stdin, os.Stdout)
		}
		if err != nil {
			printFatalln(cmd, err)
		}
	},
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	log.SetOutput(ioutil.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		debug, err := os.Create("filt.debug")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		log.SetOutput(debug)
	}

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// https://github.com/spf13/cobra/pull/894
func printErrln(c *cobra.Command, i ...interface{}) {
	c.PrintErr(fmt.Sprintln(i...))
}

func printErrf(c *cobra.Command, format string, i ...interface{}) {
	c.PrintErr(fmt.Sprintf(format, i...))
}

func printFatalln(c *cobra.Command, i ...interface{}) {
	printErrln(c, i...)
	os.Exit(1)
}

func printFatalf(c *cobra.Command, format string, i ...interface{}) {
	printErrf(c, format, i...)
	os.Exit(1)
}

func init() {
	cobra.OnInitialize(config.Load)
	rootCmd.Flags().BoolVarP(&buffered, "buffered", "b", false, "filter buffered STDIN")
	rootCmd.Flags().BoolP("version", "v", false, "print the version")
	rootCmd.SetUsageTemplate(usageTemplate)
}
