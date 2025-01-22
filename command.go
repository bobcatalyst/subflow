package subflow

import "fmt"

type Command interface {
    Command() string
}

type CommandArgs interface {
    Command
    Args() []string
}

type CommandEnv interface {
    Command
    Environment() []string
}

type CommandArgsEnv interface {
    CommandArgs
    Environment() []string
}

type basicCommandArgs struct {
    command string
    args    []string
    env     []string
}

func NewCommand(command string) Command {
    return &basicCommandArgs{
        command: command,
    }
}

func NewCommandArgs(command string, args []string) CommandArgs {
    return &basicCommandArgs{
        command: command,
        args:    args,
    }
}

func NewCommandEnv(command string, env []string) CommandEnv {
    return &basicCommandArgs{
        command: command,
        env:     env,
    }
}

func NewCommandArgsEnv(command string, args, env []string) CommandArgsEnv {
    return &basicCommandArgs{
        command: command,
        args:    args,
        env:     env,
    }
}

// WithEnv appends new environment variables to the command.
func WithEnv(cmd Command, env []string) CommandEnv {
    command, args, subEnv := commandCollect(cmd)
    return &basicCommandArgs{
        command: command,
        args:    args,
        env:     append(subEnv, env...),
    }
}

func commandCollect(cmd Command) (command string, args, env []string) {
    command = cmd.Command()
    if cmd, ok := cmd.(CommandArgs); ok {
        args = cmd.Args()
    }
    if cmd, ok := cmd.(CommandEnv); ok {
        args = cmd.Environment()
    }
    return
}

func (cmd *basicCommandArgs) Command() string       { return cmd.command }
func (cmd *basicCommandArgs) Args() []string        { return cmd.args }
func (cmd *basicCommandArgs) Environment() []string { return cmd.env }

// ErrExitCode represents a non zero process exit code.
type ErrExitCode int

func (err ErrExitCode) Error() string {
    return fmt.Sprintf("exit code(%d)", err)
}
