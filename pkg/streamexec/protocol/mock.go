package protocol

import (
	"fmt"
	"io"
	"time"
)

// MockProtocol implements Protocol for testing
type MockProtocol struct {
	ReceivedMessages []struct {
		Type MessageType
		Data []byte
	}
	CommandsToReturn   []string
	CommandReadIndex   int
	ReturnErrorOnWrite bool
	ReturnErrorOnRead  bool
	MessageDelayMs     int
}

func NewMockProtocol() *MockProtocol {
	return &MockProtocol{
		CommandsToReturn:   []string{},
		CommandReadIndex:   0,
		ReturnErrorOnWrite: false,
		ReturnErrorOnRead:  false,
		MessageDelayMs:     0,
		ReceivedMessages: []struct {
			Type MessageType
			Data []byte
		}{},
	}
}

func (p *MockProtocol) ReadMessage() (MessageType, []byte, error) {
	if p.ReturnErrorOnRead {
		return 0, nil, fmt.Errorf("mock read error")
	}

	if p.CommandReadIndex >= len(p.CommandsToReturn) {
		return 0, nil, io.EOF
	}

	// Optional delay to simulate network latency
	if p.MessageDelayMs > 0 {
		time.Sleep(time.Duration(p.MessageDelayMs) * time.Millisecond)
	}

	cmd := p.CommandsToReturn[p.CommandReadIndex]
	p.CommandReadIndex++

	return Command, []byte(cmd), nil
}

func (p *MockProtocol) WriteMessage(msgType MessageType, data []byte) error {
	if p.ReturnErrorOnWrite {
		return fmt.Errorf("mock write error")
	}

	p.ReceivedMessages = append(p.ReceivedMessages, struct {
		Type MessageType
		Data []byte
	}{
		Type: msgType,
		Data: data,
	})

	return nil
}

func (p *MockProtocol) Close() error {
	return nil
}
