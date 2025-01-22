package subflow

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "os/exec"
)

// Output struct captures the result of asynchronously running a command.
// It includes the standard output, standard error, exit code, and any execution error.
type Output struct {
    stdout, stderr []byte
    code           int
    err            error
}

// Run executes a command with the provided context and optional standard input.
func Run(ctx context.Context, cmd Command, stdin []byte) (out Output) {
    command, args, env := commandCollect(cmd)
    // Prepare the command with its context, command name, and arguments.
    c := exec.CommandContext(ctx, command, args...)
    // Set the environment variables for the command.
    c.Env = env
    // Buffers to capture standard output and standard error streams.
    var stdout, stderr bytes.Buffer
    c.Stdout, c.Stderr = &stdout, &stderr
    // Set standard input for the command
    c.Stdin = bytes.NewReader(stdin)
    // Execute the command and capture any errors.
    out.err = c.Run()
    // Populate the Output struct with the results of execution.
    out.stdout = stdout.Bytes()
    out.stderr = stderr.Bytes()
    out.code = c.ProcessState.ExitCode()
    // If there is a non-zero exit code or an error, set the error field in Output.
    if out.code != 0 {
        out.err = errors.Join(out.err, ErrExitCode(out.code))
    }
    if out.err != nil {
        out.err = fmt.Errorf("stderr(%q), %w", out.stderr, out.err)
    }
    return out
}

// Stdout returns the standard output captured during command execution.
func (out *Output) Stdout() []byte {
    return out.stdout
}

// Stderr returns the standard error captured during command execution.
func (out *Output) Stderr() []byte {
    return out.stderr
}

// Code returns the exit code of the executed command.
func (out *Output) Code() int {
    return out.code
}

// Err returns any error encountered during command execution.
func (out *Output) Err() error {
    return out.err
}
