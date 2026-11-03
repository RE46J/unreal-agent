package tui

import (
	"bufio"
	"encoding/base64"
	"io"
	"strings"
)

// Keep the render loop's exclusive output ownership while encoding directly
// into bounded buffers, rather than allocating the entire OSC 52 payload.
func writeClipboard(w io.Writer, text string) error {
	out := bufio.NewWriterSize(w, 32*1024)
	if _, err := out.WriteString("\x1b]52;c;"); err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, out)
	// Hide strings.Reader.WriteTo, which would convert the entire string to
	// []byte in one allocation when writing to the base64 encoder.
	source := struct{ io.Reader }{strings.NewReader(text)}
	if _, err := io.CopyBuffer(encoder, source, make([]byte, 24*1024)); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := out.WriteByte('\x07'); err != nil {
		return err
	}
	return out.Flush()
}

// clipboardSequence asks the terminal to put text on the system clipboard
// (OSC 52). Windows Terminal and most modern terminals honor it, including
// through a remote shell such as sbx exec, so no local clipboard tool is needed.
func clipboardSequence(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}
