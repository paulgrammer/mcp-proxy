package proxy

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Prompt represents the final configured prompt
type Prompt struct{}

// NewPrompt creates a prompt
func NewPrompt() *Prompt {
	return &Prompt{}
}

func (p *Prompt) Prompt() mcp.Prompt {
	return mcp.NewPrompt(
		p.Name(),
		// mcp.WithArgument() TODO: add arguments
		mcp.WithPromptDescription(p.Description()),
	)
}

func (p *Prompt) Name() string {
	return "Not implemented"
}

func (p *Prompt) Type() string {
	return "Not implemented"
}

func (p *Prompt) Description() string {
	return "Not implemented"
}
