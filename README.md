# Subflow

Subflow is a Go library designed for managing and executing subprocesses. It provides an interface to define commands, manage input/output streams, and handle subprocess lifecycle events synchronously and asynchronously.

---

## Installation

Install Subflow using `go get`:

```sh
go get github.com/bobcatalyst/subflow
```

Import it in your Go code:

```go
import "github.com/bobcatalyst/subflow"
```

---

## Quick Start

### Define a Command

Create a basic command:

```go
cmd := subflow.NewCommand("ls")
```

Add arguments and environment variables:

```go
cmdArgsEnv := subflow.NewCommandArgsEnv("ls", []string{"-l", "-a"}, []string{"PATH=/usr/bin"})
```

---

### Run a Command

Execute a command and get output:

```go
ctx := context.Background()
output := subflow.Run(ctx, cmdArgsEnv, nil)
fmt.Printf("Stdout: %s\n", string(output.Stdout()))
fmt.Printf("Stderr: %s\n", string(output.Stderr()))
```

---

### Manage Subprocesses

Asynchronous subprocess management:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

subCmd, err := subflow.New(ctx, cmdArgsEnv)
if err != nil {
    log.Fatalf("failed to create command: %v", err)
}
defer subCmd.Close()

go func() {
    for msg := range subCmd.Listen(ctx) {
        fmt.Printf("Message: %v\n", msg)
    }
}()

subCmd.Start()
<-subCmd.Wait()
```

---

### Streaming Input

Push input data to a running command:

```go
subCmd.Push(subflow.NewInputln("example input"))
```
