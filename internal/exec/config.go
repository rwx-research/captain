package exec

import "io"

// CommandConfig configures a command for execution
type CommandConfig struct {
	Args   []string
	Env    []string
	Name   string
	Stderr io.Writer
	Stdout io.Writer
}
