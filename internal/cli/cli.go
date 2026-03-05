package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/roniel/todo-app/internal/config"
	"github.com/roniel/todo-app/internal/focus"
	"github.com/roniel/todo-app/internal/journal"
	"github.com/roniel/todo-app/internal/task"
	"github.com/spf13/cobra"
)

// CLI holds shared state for all subcommands.
type CLI struct {
	taskStore    *task.Store
	journalStore *journal.Store
	focusStore   *focus.Store
	cfg          config.Config
	format       string
	quiet        bool
	noColor      bool
}

// printer returns a Printer writing to w using the current CLI settings.
func (c *CLI) printer(w io.Writer) *Printer {
	return newPrinter(w, c.format, c.quiet, c.noColor)
}

// New builds and returns the Cobra root command.
func New(ts *task.Store, js *journal.Store, fs *focus.Store, cfg config.Config) *cobra.Command {
	c := &CLI{
		taskStore:    ts,
		journalStore: js,
		focusStore:   fs,
		cfg:          cfg,
	}

	var useJSON bool

	root := &cobra.Command{
		Use:           "rondo",
		Short:         "RonDO — terminal productivity app",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("no subcommand provided. Run 'rondo --help' for usage")
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if useJSON {
				if cmd.Flags().Changed("format") {
					return fmt.Errorf("--json and --format cannot be used together")
				}
				c.format = "json"
			}
			// Auto-disable color when stdout is not a terminal,
			// but only if the user hasn't explicitly set --no-color.
			if !cmd.Flags().Changed("no-color") && !isTTY(os.Stdout) {
				c.noColor = true
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&c.format, "format", "table", "Output format: table, json, plain")
	root.PersistentFlags().BoolVarP(&c.quiet, "quiet", "q", false, "Suppress non-essential output")
	root.PersistentFlags().BoolVar(&c.noColor, "no-color", false, "Disable ANSI color output")
	root.PersistentFlags().BoolVar(&useJSON, "json", false, "Shorthand for --format json")

	root.AddCommand(c.addCmd())
	root.AddCommand(c.doneCmd())
	root.AddCommand(c.listCmd())
	root.AddCommand(c.showCmd())
	root.AddCommand(c.editCmd())
	root.AddCommand(c.deleteCmd())
	root.AddCommand(c.statusCmd())
	root.AddCommand(c.journalCmd())
	root.AddCommand(c.exportCmd())
	root.AddCommand(c.subtaskCmd())
	root.AddCommand(c.timelogCmd())
	root.AddCommand(c.recurCmd())
	root.AddCommand(c.configCmd())
	root.AddCommand(c.statsCmd())
	root.AddCommand(c.focusCmd())
	root.AddCommand(c.noteCmd())
	root.AddCommand(c.batchCmd())
	root.AddCommand(c.completionCmd())

	return root
}

// Run is the CLI entry point.
func Run(args []string, ts *task.Store, js *journal.Store, fs *focus.Store, cfg config.Config) error {
	root := New(ts, js, fs, cfg)
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	return root.Execute()
}
