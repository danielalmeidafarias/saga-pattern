package msg

import "encoding/json"

type MessageType string

// Failure is safe to serialize and represents a business failure reported by a consumer.
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Message is the transport envelope shared by every service.
type Message struct {
	ID      string          `json:"id"`
	Topic   string          `json:"topic"`
	SagaID  string          `json:"sagaId"`
	StepID  string          `json:"stepId"`
	OrderID string          `json:"orderId"`
	Type    MessageType     `json:"type"`
	Version int             `json:"version"`
	Payload json.RawMessage `json:"payload"`
	Failure *Failure        `json:"failure,omitempty"`
}

func NewMessage(topic string, messageType MessageType, payload json.RawMessage) Message {
	return Message{
		Topic:   topic,
		Type:    messageType,
		Version: 1,
		Payload: payload,
	}
}
