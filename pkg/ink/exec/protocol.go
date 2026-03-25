package exec

import (
	"context"
	"encoding/json"
	"fmt"

	"nhooyr.io/websocket"
)

const (
	chanStdin  = 0x00
	chanStdout = 0x01
	chanStderr = 0x02
	chanResize = 0x03
)

func writeFrame(ctx context.Context, conn *websocket.Conn, channel byte, data []byte) error {
	frame := make([]byte, 1+len(data))
	frame[0] = channel
	copy(frame[1:], data)
	return conn.Write(ctx, websocket.MessageBinary, frame)
}

func sendResize(ctx context.Context, conn *websocket.Conn, width, height uint16) error {
	payload, err := json.Marshal(struct {
		Width  uint16 `json:"width"`
		Height uint16 `json:"height"`
	}{width, height})
	if err != nil {
		return fmt.Errorf("exec: marshal resize: %w", err)
	}
	return writeFrame(ctx, conn, chanResize, payload)
}
