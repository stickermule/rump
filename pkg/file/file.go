// Package file allows reading/writing from/to a Rump file.
package file

import (
	"context"
	"bufio"
	"fmt"
	"os"

	"github.com/stickermule/rump/pkg/message"
)

// File can read and write, to a file Path, using the message Bus.
type File struct {
	Path string
	Bus  message.Bus
}

// New creates the File struct, to be used for reading/writing.
func New(path string, bus message.Bus) *File {
	return &File{
		Path: path,
		Bus:  bus,
	}
}

// Read scans a Rump .dump file and sends Payloads to the message bus.
func (f *File) Read(ctx context.Context) error {
	defer close(f.Bus)
	d, err := os.Open(f.Path)
	defer d.Close()
	if err != nil {
		return err
	}

	// Scan file with default ScanLines
	scanner := bufio.NewScanner(d)

	// Scan line by line
	// file protocol is two lines per key/value pair: key\n value\n
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("exiting")
			return ctx.Err()
		default:
			// Get key on first line
			key := scanner.Text()
			// trigger next scan to get value on next line
			scanner.Scan()
			value := scanner.Text()
			f.Bus <- message.Payload{Key: key, Value: value}
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
			fmt.Println("exiting")
			return ctx.Err()
		// Get Messages from Bus
		case p, ok := <-f.Bus:
			// if channel closed, set to nil, break loop
			if !ok {
				f.Bus = nil
				continue
			}
			_, err := w.WriteString(p.Key + "\n" + p.Value + "\n")
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
