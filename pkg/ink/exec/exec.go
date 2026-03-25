// Package exec provides interactive shell sessions to Ink services via WebSocket.
package exec

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"

	ink "github.com/mldotink/cli/pkg/ink"
	"nhooyr.io/websocket"
)

// Session is an interactive shell session connected to a running service container.
type Session struct {
	conn    *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	stdoutR *io.PipeReader
	stdoutW *io.PipeWriter
	stderrR *io.PipeReader
	stderrW *io.PipeWriter
	mu      sync.Mutex
}

// Dial obtains an exec token via the Ink API and opens a WebSocket shell
// session to the specified service.
func Dial(ctx context.Context, client *ink.Client, serviceID string) (*Session, error) {
	sess, err := client.ExecURL(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("exec: get session: %w", err)
	}

	wsURL := sess.URL + "?token=" + url.QueryEscape(sess.Token)
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("exec: dial: %w", err)
	}
	conn.SetReadLimit(1 << 20) // 1 MiB

	sessionCtx, cancel := context.WithCancel(ctx)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	s := &Session{
		conn:    conn,
		ctx:     sessionCtx,
		cancel:  cancel,
		stdoutR: stdoutR,
		stdoutW: stdoutW,
		stderrR: stderrR,
		stderrW: stderrW,
	}

	s.wg.Add(1)
	go s.readLoop()

	return s, nil
}

func (s *Session) readLoop() {
	defer s.wg.Done()
	defer s.stdoutW.Close()
	defer s.stderrW.Close()

	for {
		_, msg, err := s.conn.Read(s.ctx)
		if err != nil {
			return
		}
		if len(msg) < 2 {
			continue
		}
		switch msg[0] {
		case chanStdout:
			s.stdoutW.Write(msg[1:])
		case chanStderr:
			s.stderrW.Write(msg[1:])
		}
	}
}

// Stdin returns a writer for sending input to the shell.
func (s *Session) Stdin() io.Writer { return &stdinWriter{s: s} }

// Stdout returns a reader for the shell's stdout stream.
func (s *Session) Stdout() io.Reader { return s.stdoutR }

// Stderr returns a reader for the shell's stderr stream.
func (s *Session) Stderr() io.Reader { return s.stderrR }

// Resize sends a terminal resize event to the remote shell.
func (s *Session) Resize(width, height uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return sendResize(s.ctx, s.conn, width, height)
}

// Wait blocks until the session ends. Returns nil on clean close.
func (s *Session) Wait() error {
	s.wg.Wait()
	return nil
}

// Close terminates the shell session.
func (s *Session) Close() error {
	s.cancel()
	return s.conn.Close(websocket.StatusNormalClosure, "")
}

type stdinWriter struct {
	s *Session
}

func (w *stdinWriter) Write(p []byte) (int, error) {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	if err := writeFrame(w.s.ctx, w.s.conn, chanStdin, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
