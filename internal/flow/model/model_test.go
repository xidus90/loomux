package model_test

import (
	"context"

	"github.com/xidus90/loomux/internal/flow/model"
)

// A double for the port, so a moved signature breaks the build here.
type stubModel struct{}

func (stubModel) Ask(context.Context, model.Request) (model.Reply, error) {
	return model.Reply{}, nil
}

var _ model.Model = stubModel{}
