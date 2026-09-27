// Package model is the port every agent node reaches a model through. Tests
// hand the runner the Fake; the adapters for the providers' CLIs arrive later.
package model

import "context"

// ReplyType is the type of one field a reply must carry.
type ReplyType string

// The three types a reply field can have; replies are flat on purpose.
const (
	ReplyString ReplyType = "string"
	ReplyInt    ReplyType = "int"
	ReplyBool   ReplyType = "bool"
)

// Request is one model call, fully described.
type Request struct {
	Prompt   string
	Tools    []string
	Effort   string
	Provider string
	Model    string               // empty: the provider CLI's own default
	Reply    map[string]ReplyType // every field the answer must carry, and no other
}

// Reply is a model's answer and what it cost.
type Reply struct {
	Fields map[string]any // a string, int or bool per field of Request.Reply
	Tokens int
	Model  string // the model that actually answered; empty when the adapter cannot tell
}

// Model answers requests.
type Model interface {
	Ask(ctx context.Context, request Request) (Reply, error)
}
