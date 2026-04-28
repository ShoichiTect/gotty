package webtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"sync"
	"testing"
)

type pipePair struct {
	*io.PipeReader
	*io.PipeWriter
}

type pipeSlave struct {
	pipePair
}

func (slave pipeSlave) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (slave pipeSlave) ResizeTerminal(columns int, rows int) error {
	return nil
}

func runWebTTY(t *testing.T, wt *WebTTY, ctx context.Context) *sync.WaitGroup {
	t.Helper()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := wt.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && err != ErrMasterClosed && err != ErrSlaveClosed {
			t.Errorf("Unexpected error from Run(): %s", err)
		}
	}()
	return &wg
}

func readMessage(t *testing.T, reader io.Reader, want byte) []byte {
	t.Helper()

	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("Unexpected error from Read(): %s", err)
		}
		if n == 0 {
			continue
		}
		if buf[0] == want {
			message := make([]byte, n)
			copy(message, buf[:n])
			return message
		}
	}
}

func TestWriteFromPTY(t *testing.T) {
	connInPipeReader, connInPipeWriter := io.Pipe()     // in to conn
	connOutPipeReader, connOutPipeWriter := io.Pipe()   // out from conn
	slaveInPipeReader, slaveInPipeWriter := io.Pipe()   // in to slave
	slaveOutPipeReader, slaveOutPipeWriter := io.Pipe() // out from slave

	conn := pipePair{
		connOutPipeReader,
		connInPipeWriter,
	}
	slave := pipeSlave{pipePair{
		slaveOutPipeReader,
		slaveInPipeWriter,
	}}
	dt, err := New(conn, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := runWebTTY(t, dt, ctx)
	defer func() {
		cancel()
		connOutPipeWriter.Close()
		slaveOutPipeWriter.Close()
		connInPipeReader.Close()
		slaveInPipeReader.Close()
		wg.Wait()
	}()
	readMessage(t, connInPipeReader, SetWindowTitle)

	message := []byte("foobar")
	n, err := slaveOutPipeWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	got := readMessage(t, connInPipeReader, Output)
	decoded := make([]byte, 1024)
	n, err = base64.StdEncoding.Decode(decoded, got[1:])
	if err != nil {
		t.Fatalf("Unexpected error from Decode(): %s", err)
	}
	if !bytes.Equal(decoded[:n], message) {
		t.Fatalf("Unexpected message received: `%s`", decoded[:n])
	}

	_ = slaveInPipeReader.Close()
}

func TestWriteFromConn(t *testing.T) {
	connInPipeReader, connInPipeWriter := io.Pipe()     // in to conn
	connOutPipeReader, connOutPipeWriter := io.Pipe()   // out from conn
	slaveInPipeReader, slaveInPipeWriter := io.Pipe()   // in to slave
	slaveOutPipeReader, slaveOutPipeWriter := io.Pipe() // out from slave

	conn := pipePair{
		connOutPipeReader,
		connInPipeWriter,
	}
	slave := pipeSlave{pipePair{
		slaveOutPipeReader,
		slaveInPipeWriter,
	}}
	dt, err := New(conn, slave, WithPermitWrite())
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := runWebTTY(t, dt, ctx)
	defer func() {
		cancel()
		connOutPipeWriter.Close()
		slaveOutPipeWriter.Close()
		connInPipeReader.Close()
		slaveInPipeReader.Close()
		wg.Wait()
	}()
	readMessage(t, connInPipeReader, SetWindowTitle)

	// input
	message := []byte{Input, 'h', 'e', 'l', 'l', 'o', '\n'}
	n, err := connOutPipeWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	readBuf := make([]byte, 1024)
	n, err = slaveInPipeReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error from Read(): %s", err)
	}
	if !bytes.Equal(readBuf[:n], message[1:]) {
		t.Fatalf("Unexpected message received: `%s`", readBuf[:n])
	}

	// ping
	message = []byte{Ping}
	n, err = connOutPipeWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	got := readMessage(t, connInPipeReader, Pong)
	if !bytes.Equal(got, []byte{Pong}) {
		t.Fatalf("Unexpected message received: `%s`", got)
	}
}
