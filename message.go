package subflow

import (
    "encoding/json"
    "fmt"
    "reflect"
    "slices"
    "time"
)

// Message represents a generic interface for message types.
type Message interface {
    message()
}

// Input extends Message and provides a method to get the input data as bytes.
type Input interface {
    Message
    Input() []byte
}

type BaseMessage[K fmt.Stringer] struct {
    Time time.Time     `json:"time"`
    Kind JSONString[K] `json:"kind"`
}

// NewBaseMessage initializes a new BaseMessage with the current time.
func NewBaseMessage[K fmt.Stringer]() BaseMessage[K] {
    return BaseMessage[K]{Time: time.Now()}
}

func (BaseMessage[K]) message() {}

// JSONString wraps a type that implements fmt.Stringer for JSON serialization.
type JSONString[S fmt.Stringer] struct{}

func (JSONString[S]) String() string { return (*new(S)).String() }

func (js JSONString[S]) MarshalJSON() ([]byte, error) {
    return json.Marshal(js.String())
}

func (js *JSONString[S]) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    } else if s != js.String() {
        return fmt.Errorf("expected static string %q, got %q", js.String(), s)
    }
    return nil
}

// Data represents a slice of bytes that can be serialized to/from JSON.
type Data []byte

func (d Data) MarshalJSON() ([]byte, error) {
    return json.Marshal(string(d))
}

func (d *Data) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    *d = []byte(s)
    return nil
}

// DataLike defines types that behave like Data (string or []byte).
type DataLike interface {
    ~string | ~[]byte
}

type kind[K any] struct{}

func (kind[K]) String() string {
    return reflect.TypeFor[K]().Name()
}

type (
    stdio  struct{}
    start  struct{}
    exit   struct{}
    stderr struct{}
    stdout struct{}
    stdin  struct{}
    text   struct{}
)

type (
    // StartMessage represents a message indicating the start of a process.
    StartMessage struct {
        BaseMessage[kind[start]]
    }

    // ExitMessage represents a message indicating the end of a process, including the exit code.
    ExitMessage struct {
        BaseMessage[kind[exit]]
        Code int `json:"code"`
    }
)

func NewStartMessage() Message {
    return StartMessage{BaseMessage: NewBaseMessage[kind[start]]()}
}

func NewExitMessage(code int) Message {
    return ExitMessage{
        BaseMessage: NewBaseMessage[kind[exit]](),
        Code:        code,
    }
}

type (
    stdioMessage[K fmt.Stringer] struct {
        BaseMessage[kind[stdio]]
        Stdio JSONString[K] `json:"stdio"`
        Data  Data          `json:"data"`
    }
    StdinMessage  = stdioMessage[kind[stdin]]
    StderrMessage = stdioMessage[kind[stderr]]
    StdoutMessage = stdioMessage[kind[stdout]]
)

func newStdioMessage[K fmt.Stringer, D DataLike](data D) stdioMessage[K] {
    return stdioMessage[K]{
        BaseMessage: NewBaseMessage[kind[stdio]](),
        Data:        slices.Clone(Data(data)),
    }
}

// StdioLike groups StdioMessage types for stdin, stdout, and stderr.
type StdioLike interface {
    StderrMessage | StdoutMessage | StdinMessage
}

// NewStdioMessage creates a specific type of StdioMessage based on the provided data.
func NewStdioMessage[T StdioLike, D DataLike](data D) Message {
    var msg T
    switch msg := any(&msg).(type) {
    case *StderrMessage:
        *msg = newStdioMessage[kind[stderr]](data)
    case *StdoutMessage:
        *msg = newStdioMessage[kind[stdout]](data)
    case *StdinMessage:
        *msg = newStdioMessage[kind[stdin]](data)
    default:
        panic("invalid stdio type")
    }
    return any(msg).(Message)
}

// TextInput represents input data as a message.
type TextInput struct {
    BaseMessage[kind[text]]
    Data Data `json:"data"`
}

func (ti TextInput) Input() []byte {
    return ti.Data
}

func newTextInput[D DataLike](data D) TextInput { return TextInput{Data: []byte(data)} }

// NewInputln creates a new TextInput with a newline appended.
func NewInputln[D DataLike](data D) Input {
    return newTextInput(append(slices.Clone([]byte(data)), '\n'))
}

// NewInput creates a new TextInput.
func NewInput[D DataLike](data D) Input { return newTextInput(slices.Clone([]byte(data))) }

// NewInputf creates a new TextInput with formatted data.
func NewInputf(format string, a ...any) Input { return newTextInput(fmt.Sprintf(format, a...)) }
