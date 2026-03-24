package cobra

import (
	"flag"
	"fmt"
	"io"
)

// PositionalArgs validates non-flag command arguments.
type PositionalArgs func(cmd *Command, args []string) error

// Command is a small Cobra-compatible command wrapper used by this project.
type Command struct {
	Use   string
	Short string

	Args PositionalArgs
	Run  func(cmd *Command, args []string)
	RunE func(cmd *Command, args []string) error

	SilenceErrors bool
	SilenceUsage  bool

	args  []string
	flags *flag.FlagSet
}

// NoArgs rejects any positional arguments.
func NoArgs(cmd *Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("%s does not accept arguments", cmd.Use)
	}
	return nil
}

// Flags returns the command flag set.
func (c *Command) Flags() *flag.FlagSet {
	if c.flags == nil {
		fs := flag.NewFlagSet(c.Use, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		c.flags = fs
	}
	return c.flags
}

// SetArgs stores the raw command arguments for later parsing.
func (c *Command) SetArgs(args []string) {
	c.args = append([]string(nil), args...)
}

// ExecuteC parses flags and runs the command.
func (c *Command) ExecuteC() (*Command, error) {
	fs := c.Flags()
	if err := fs.Parse(c.args); err != nil {
		return c, err
	}

	remaining := fs.Args()
	if c.Args != nil {
		if err := c.Args(c, remaining); err != nil {
			return c, err
		}
	}

	if c.RunE != nil {
		if err := c.RunE(c, remaining); err != nil {
			return c, err
		}
		return c, nil
	}

	if c.Run != nil {
		c.Run(c, remaining)
	}

	return c, nil
}
