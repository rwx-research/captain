package exec

// Command is a generic interface that represents a command that is being executed. This is modelled after the default
// `exec.Cmd` from the `os/exec` package.
type Command interface {
	Start() error
	Wait() error
}
