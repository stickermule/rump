// Package message represents the message bus.
// Message Payloads pass through a Bus channel.
package message

// Payload represents a Redis key/value pair.
type Payload struct {
	Key   string
	Value string
}

// Bus is a channel where message Payloads pass.
type Bus chan Payload
