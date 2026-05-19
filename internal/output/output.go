// Package output centralises how the CLI prints JSON results and errors.
//
// We pretty-print successful responses with encoding/json's MarshalIndent
// using two-space indentation. Errors are wrapped in ANSI red when the
// writer is a TTY; otherwise the codes are dropped so logs stay clean.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI escape codes. Only emitted when the destination is a TTY.
const (
	ansiReset = "\x1b[0m"
	ansiRed   = "\x1b[31m"
	ansiBold  = "\x1b[1m"
)

// Printer formats command results to a writer. Construct one per
// process; it is safe for concurrent use as long as the underlying
// writer is.
type Printer struct {
	out   io.Writer
	err   io.Writer
	color bool
}

// New returns a Printer that writes to out (success) and err (errors).
// Colorisation is enabled when err is connected to a terminal.
func New(out, err io.Writer) *Printer {
	return &Printer{out: out, err: err, color: isTerminal(err)}
}

// NewWithColor lets tests opt in or out of colour explicitly.
func NewWithColor(out, err io.Writer, color bool) *Printer {
	return &Printer{out: out, err: err, color: color}
}

// JSON pretty-prints v to the success writer.
func (p *Printer) JSON(v any) error {
	if v == nil {
		return nil
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("output: marshal: %w", err)
	}
	if _, err := fmt.Fprintln(p.out, string(b)); err != nil {
		return fmt.Errorf("output: write: %w", err)
	}
	return nil
}

// Println writes a plain message to the success writer with a trailing newline.
func (p *Printer) Println(msg string) {
	fmt.Fprintln(p.out, msg)
}

// Printf writes a formatted message to the success writer.
func (p *Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.out, format, args...)
}

// Error writes err to the error writer, colourising in red when the
// writer is a TTY.
func (p *Printer) Error(err error) {
	if err == nil {
		return
	}
	if p.color {
		fmt.Fprintf(p.err, "%s%sError:%s %s\n", ansiRed, ansiBold, ansiReset, err.Error())
		return
	}
	fmt.Fprintf(p.err, "Error: %s\n", err.Error())
}

// Errorf writes a formatted error message.
func (p *Printer) Errorf(format string, args ...any) {
	p.Error(fmt.Errorf(format, args...))
}

// Color reports whether colour codes are emitted. Useful for tests.
func (p *Printer) Color() bool { return p.color }

// isTerminal returns true when w is *os.File pointing at a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
