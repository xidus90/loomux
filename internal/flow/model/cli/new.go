package cli

import (
	"fmt"

	"github.com/xidus90/loomux/internal/flow/model"
)

// New returns the adapter for a provider's CLI.
func New(provider string, o Options) (model.Model, error) {
	switch provider {
	case "claude":
		return NewClaude(o), nil
	case "agy":
		return NewAgy(o), nil
	case "qoder":
		return NewQoder(o), nil
	}
	return nil, fmt.Errorf("no adapter for provider %s", provider)
}
