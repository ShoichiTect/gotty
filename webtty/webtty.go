package webtty

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// WebTTY bridges a PTY slave and its PTY master.
// To support text-based streams and side channel commands such as
// terminal resizing, WebTTY uses an original protocol.
type WebTTY struct {
	// PTY Master, which probably a connection to browser
	masterConn Master
	// PTY Slave
	slave Slave

	windowTitle  []byte
	permitWrite  bool
	columns      int
	rows         int
	reconnect    int // in seconds
	masterPrefs  []byte
	eventHandler func(event string, fields map[string]interface{})

	bufferSize int
	writeMutex sync.Mutex
}

// New creates a new instance of WebTTY.
// masterConn is a connection to the PTY master,
// typically it's a websocket connection to a client.
// slave is a PTY slave such as a local command with a PTY.
func New(masterConn Master, slave Slave, options ...Option) (*WebTTY, error) {
	wt := &WebTTY{
		masterConn: masterConn,
		slave:      slave,

		permitWrite: false,
		columns:     0,
		rows:        0,

		bufferSize: 1024,
	}

	for _, option := range options {
		option(wt)
	}

	return wt, nil
}

// Run starts the main process of the WebTTY.
// This method blocks until the context is canceled.
// Note that the master and slave are left intact even
// after the context is canceled. Closing them is caller's
// responsibility.
// If the connection to one end gets closed, returns ErrSlaveClosed or ErrMasterClosed.
func (wt *WebTTY) Run(ctx context.Context) error {
	err := wt.sendInitializeMessage()
	if err != nil {
		return fmt.Errorf("failed to send initializing message: %w", err)
	}

	errs := make(chan error, 2)

	go func() {
		errs <- func() error {
			buffer := make([]byte, wt.bufferSize)
			for {
				n, err := wt.slave.Read(buffer)
				if err != nil {
					return ErrSlaveClosed
				}

				err = wt.handleSlaveReadEvent(buffer[:n])
				if err != nil {
					return err
				}
			}
		}()
	}()

	go func() {
		errs <- func() error {
			buffer := make([]byte, wt.bufferSize)
			for {
				n, err := wt.masterConn.Read(buffer)
				if err != nil {
					return ErrMasterClosed
				}

				err = wt.handleMasterReadEvent(buffer[:n])
				if err != nil {
					return err
				}
			}
		}()
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errs:
	}

	return err
}

func (wt *WebTTY) sendInitializeMessage() error {
	wt.emitEvent("initialize", map[string]interface{}{
		"reconnect":     wt.reconnect,
		"fixed_columns": wt.columns,
		"fixed_rows":    wt.rows,
		"permit_write":  wt.permitWrite,
	})

	err := wt.masterWrite(append([]byte{SetWindowTitle}, wt.windowTitle...))
	if err != nil {
		return fmt.Errorf("failed to send window title: %w", err)
	}

	if wt.reconnect > 0 {
		reconnect, _ := json.Marshal(wt.reconnect)
		err := wt.masterWrite(append([]byte{SetReconnect}, reconnect...))
		if err != nil {
			return fmt.Errorf("failed to set reconnect: %w", err)
		}
	}

	if wt.masterPrefs != nil {
		err := wt.masterWrite(append([]byte{SetPreferences}, wt.masterPrefs...))
		if err != nil {
			return fmt.Errorf("failed to set preferences: %w", err)
		}
	}

	return nil
}

func (wt *WebTTY) handleSlaveReadEvent(data []byte) error {
	safeMessage := base64.StdEncoding.EncodeToString(data)
	err := wt.masterWrite(append([]byte{Output}, []byte(safeMessage)...))
	if err != nil {
		return fmt.Errorf("failed to send message to master: %w", err)
	}

	return nil
}

func (wt *WebTTY) masterWrite(data []byte) error {
	wt.writeMutex.Lock()
	defer wt.writeMutex.Unlock()

	_, err := wt.masterConn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to master: %w", err)
	}

	return nil
}

func (wt *WebTTY) handleMasterReadEvent(data []byte) error {
	if len(data) == 0 {
		return errors.New("unexpected zero length read from master")
	}

	switch data[0] {
	case Input:
		wt.emitEvent("input", map[string]interface{}{
			"bytes":        len(data) - 1,
			"permit_write": wt.permitWrite,
			"hex":          fmt.Sprintf("%x", data[1:]),
		})
		if !wt.permitWrite {
			return nil
		}

		if len(data) <= 1 {
			return nil
		}

		_, err := wt.slave.Write(data[1:])
		if err != nil {
			return fmt.Errorf("failed to write received data to slave: %w", err)
		}

	case Ping:
		wt.emitEvent("ping", nil)
		err := wt.masterWrite([]byte{Pong})
		if err != nil {
			return fmt.Errorf("failed to return Pong message to master: %w", err)
		}

	case InputDebug:
		wt.emitEvent("input_debug", map[string]interface{}{
			"payload": string(data[1:]),
		})

	case ResizeTerminal:
		if wt.columns != 0 && wt.rows != 0 {
			break
		}

		if len(data) <= 1 {
			return errors.New("received malformed remote command for terminal resize: empty payload")
		}

		var args argResizeTerminal
		err := json.Unmarshal(data[1:], &args)
		if err != nil {
			return fmt.Errorf("received malformed data for terminal resize: %w", err)
		}
		rows := wt.rows
		if rows == 0 {
			rows = int(args.Rows)
		}

		columns := wt.columns
		if columns == 0 {
			columns = int(args.Columns)
		}

		wt.emitEvent("resize", map[string]interface{}{
			"columns": columns,
			"rows":    rows,
		})
		wt.slave.ResizeTerminal(columns, rows)
	default:
		return fmt.Errorf("unknown message type `%c`", data[0])
	}

	return nil
}

func (wt *WebTTY) emitEvent(event string, fields map[string]interface{}) {
	if wt.eventHandler != nil {
		wt.eventHandler(event, fields)
	}
}

type argResizeTerminal struct {
	Columns float64
	Rows    float64
}
