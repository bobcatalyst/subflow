package subflow

import (
    "context"
    "errors"
    "github.com/bobcatalyst/flow"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "slices"
    "sync"
    "sync/atomic"
    "time"
)

type Cmd struct {
    stdin io.WriteCloser
    in    flow.Stream[Input]
    out   flow.Stream[Message]

    cmd    *exec.Cmd
    ctx    context.Context
    cancel context.CancelFunc
    stop   func() bool

    started  atomic.Bool
    wait     chan struct{}
    waitErr  error
    killOnce sync.Once
}

func New(ctx context.Context, cmd CommandArgs) (_ *Cmd, finalErr error) {
    finally, cleanup := checkOk()

    // Setup command struct
    ctx, cancel := context.WithCancel(ctx)
    defer cleanup(cancel)
    c := Cmd{
        ctx:    ctx,
        cancel: cancel,
        wait:   make(chan struct{}),
    }

    // Make command and setup io
    in, err := c.initializeCommand(cmd)
    if err != nil {
        return nil, err
    }
    c.stdin = in
    defer cleanup(func() { finalErr = errors.Join(finalErr, c.Close()) })

    // Make sure close is run at lease once if one of the goroutines cancels the context
    c.stop = context.AfterFunc(ctx, func() { c.Close() })
    defer cleanup(func() { c.stop() })

    finally()
    return &c, nil
}

func checkOk() (finally func(), cleanup func(func())) {
    ok := false
    return func() {
            ok = true
        }, func(fn func()) {
            if !ok {
                fn()
            }
        }
}

// Push adds new inputs to the command's input stream
func (cmd *Cmd) Push(in ...Input) { cmd.in.Push(in...) }

// Listen emits the process start, stdout/err/in, and the exit code.
// It is non buffered, so any messages emitted before Listen is called will be lost.
// Call Listen before Start to get all messages.
//
//	c1 := cmd.Listen(context.Background)
//	cmd.Start()
//	c2 := cmd.Listen(context.Background)
//
// c1 will contain the start message while c2 will not.
func (cmd *Cmd) Listen(ctx context.Context) <-chan Message { return cmd.out.Listen(ctx) }

// Start starts the command exactly once.
func (cmd *Cmd) Start() {
    if cmd.started.CompareAndSwap(false, true) {
        go cmd.runCmd()
    }
}

// Done returns a channel that closes when the process completes.
func (cmd *Cmd) Done() <-chan struct{} {
    return cmd.wait
}

// Close closes the Cmd waiting indefinitely for the subprocess to exit.
func (cmd *Cmd) Close() error {
    return cmd.CloseTimeout(0)
}

// CloseTimeout stops the command and cleans up resources. If the command does not terminate, it will be killed after a timeout.
func (cmd *Cmd) CloseTimeout(timeout time.Duration) error {
    cmd.cancel()
    cmd.stop()
    if cmd.started.CompareAndSwap(false, true) {
        // never started
        cmd.cleanupCmd(false)
        cmd.killOnce.Do(func() {})
    } else {
        cmd.killOnce.Do(func() {
            if timeout > 0 {
                select {
                case <-cmd.Done():
                case <-time.After(timeout):
                    _ = cmd.cmd.Process.Signal(os.Kill)
                }
            }
        })
    }
    <-cmd.Done()
    return cmd.waitErr
}

// runCmd starts and monitors the command, handling input and capturing output
func (cmd *Cmd) runCmd() {
    defer cmd.cleanupCmd(true)
    setCode, sendCode := cmd.exitCode()
    cmd.out.Push(NewStartMessage())
    defer sendCode()

    go cmd.pipeInput(cmd.in.Listen(cmd.ctx), cmd.stdin)
    if err := cmd.cmd.Run(); err != nil {
        setCode(-1)
        if exit := new(exec.ExitError); errors.As(err, &exit) {
            setCode(exit.ExitCode())
        } else {
            cmd.waitErr = errors.Join(cmd.waitErr, err)
        }
    }
}

func (cmd *Cmd) exitCode() (setCode func(code int), sendCode func()) {
    var code int
    setCode = func(c int) {
        code = c
    }
    sendCode = func() {
        if code != 0 {
            cmd.waitErr = errors.Join(cmd.waitErr, ErrExitCode(code))
        }
        cmd.out.Close(NewExitMessage(code))
    }
    return
}

func (cmd *Cmd) cleanupCmd(started bool) {
    defer close(cmd.wait)
    if !started {
        cmd.out.Close()
    }
    // cmd.stdin will not be nil
    cmd.waitErr = errors.Join(cmd.waitErr, cmd.stdin.Close())
}

func (cmd *Cmd) initializeCommand(cae Command) (stdin io.WriteCloser, _ error) {
    command, args, env := commandCollect(cae)
    cmd.cmd = exec.CommandContext(cmd.ctx, command, args...)
    if len(cmd.cmd.Env) == 0 {
        cmd.cmd.Env = os.Environ()
    }
    cmd.cmd.Env = append(cmd.cmd.Env, env...)
    cmd.cmd.Stdout, cmd.cmd.Stderr = cmd.newKindWriters()
    return cmd.cmd.StdinPipe()
}

func (cmd *Cmd) newKindWriters() (*kindWriter[StdoutMessage], *kindWriter[StderrMessage]) {
    return &kindWriter[StdoutMessage]{
            out: &cmd.out,
            ctx: cmd.ctx,
        }, &kindWriter[StderrMessage]{
            out: &cmd.out,
            ctx: cmd.ctx,
        }
}

type kindWriter[K StdioLike] struct {
    out flow.Pushable[Message]
    ctx context.Context
}

func (kw *kindWriter[K]) Write(b []byte) (n int, _ error) {
    if kw.ctx.Err() != nil {
        return 0, kw.ctx.Err()
    }
    kw.out.Push(NewStdioMessage[K](slices.Clone(b)))
    return len(b), nil
}

func (cmd *Cmd) pipeInput(stdin <-chan Input, in io.WriteCloser) {
    defer in.Close()
    defer cmd.cancel()

    for cmd.ctx.Err() == nil {
        select {
        case <-cmd.ctx.Done():
            return
        case data, ok := <-stdin:
            if ok {
                b := data.Input()
                n, err := in.Write(b)
                cmd.out.Push(NewStdioMessage[StdinMessage](b[:n]))
                if err != nil {
                    return
                } else if n <= len(b) {
                    slog.Error("incomplete write of stdin")
                }
            } else {
                return
            }
        }
    }
}
