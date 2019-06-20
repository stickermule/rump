// Package file allows reading/writing from/to a Rump file.
// Rump file protocol is key✝✝value✝✝key✝✝value✝✝...
package file

import (
	"context"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/stickermule/rump/pkg/message"
)

// File can read and write, to a file Path, using the message Bus.
type File struct {
	Path string
	Bus  message.Bus
}

// split is double-cross (✝✝) custom Scanner Split.
func splitCross(data []byte, atEOF bool) (advance int, token []byte, err error) {

	// end of file
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Split at separator
	if i := strings.Index(string(data), "✝✝"); i >= 0 {
		// Separator is 6 bytes long
		return i + 6, data[0:i], nil
	}

	return 0, nil, nil
}

// New creates the File struct, to be used for reading/writing.
func New(path string, bus message.Bus) *File {
	return &File{
		Path: path,
		Bus:  bus,
	}
}

// Read scans a Rump file and sends Payloads to the message bus.
func (f *File) Read(ctx context.Context) error {
	defer close(f.Bus)
	d, err := os.Open(f.Path)
	defer d.Close()
	if err != nil {
		return err
	}

	// Scan file, split by double-cross separator
	scanner := bufio.NewScanner(d)
	scanner.Split(splitCross)

	// Scan line by line
	// file protocol is key✝✝value✝✝
	for scanner.Scan() {
		// Get key on first line
		key := scanner.Text()
		// trigger next scan to get value on next line
		scanner.Scan()
		value := scanner.Text()
		select {
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("file read: exit")
			return ctx.Err()
		case f.Bus <- message.Payload{Key: key, Value: value}:
			fmt.Printf("r")
		}
	}

	return nil
}

// Write writes to a Rump file Payloads from the message bus.
func (f *File) Write(ctx context.Context) error {
	d, err := os.Create(f.Path)
	defer d.Close()
	if err != nil {
		return err
	}

	// Buffered write to limit system IO calls
	w := bufio.NewWriter(d)

	for f.Bus != nil {
		select {
		// Exit early if context done.
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("file write: exit")
			return ctx.Err()
		// Get Messages from Bus
		case p, ok := <-f.Bus:
			// if channel closed, set to nil, break loop
			if !ok {
				f.Bus = nil
				continue
			}
			_, err := w.WriteString(p.Key + "✝✝" + p.Value + "✝✝")
			if err != nil {
				return err
			}
			fmt.Printf("w")
		}
	}

	// Flush last open buffers
	w.Flush()

	return nil
}
