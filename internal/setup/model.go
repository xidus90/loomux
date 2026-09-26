package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
)

// The part model is the one tool init installs rather than names: every
// other tool is checked and its install command shown, while the local
// model is pulled into Ollama once the human confirms it. Only init pulls;
// reconcile asks the model that is there or counts the case as manual.

// ollama is what init asks of Ollama.
type ollama interface {
	Has(ctx context.Context, name string) (bool, error)
	Pull(ctx context.Context, name string, progress func(status string, completed, total int64)) error
}

// realOllama is the loopback client of internal/brain/model; its guard's
// refusal comes back as the error.
func realOllama(s config.ModelSettings) (ollama, error) {
	client, err := model.NewClient(s)
	if err != nil {
		// A nil *Client inside the interface would not be nil.
		return nil, err
	}
	return client, nil
}

// newOllama builds the client; tests replace it so no result depends on an
// Ollama the machine runs.
var newOllama = realOllama

// localOnly says whether the configuration text declares [privacy] mode =
// "local_only", read through the schema as DefaultChoice reads it. Text that
// does not parse declares nothing; Build refuses it on its own.
func localOnly(text string) bool {
	entries, _ := schema.Current(text)
	return slices.ContainsFunc(entries, func(e schema.Entry) bool {
		return e.Key.ID() == "privacy.mode" && e.Origin == schema.Set && e.Input == "local_only"
	})
}

// model plans the pull of the local model while Ollama lacks it. Settings
// that do not read, an endpoint the guard refuses and an Ollama that cannot
// be asked are notes: none of them stops init.
func (b *builder) model() {
	s := b.f.Model
	if b.f.ModelProblem != "" {
		b.note("model: skipped; " + b.f.ModelProblem)
		return
	}
	client, err := newOllama(s)
	if err != nil {
		b.note("model: skipped; " + err.Error())
		return
	}
	there, err := client.Has(context.Background(), s.Name)
	switch {
	case errors.Is(err, model.ErrUnreachable):
		b.note("ollama is not running at " + s.Endpoint + "; start it and run init again, or run: ollama pull " + s.Name)
	case err != nil:
		b.note("model: skipped; ollama at " + s.Endpoint + " could not be asked for " + s.Name + ": " + err.Error())
	case !there:
		b.action("model", "model-pull", "ollama pull "+s.Name)
	}
}

// PullModel pulls the model s names and writes its progress to w: a line
// per status, and for a layer being downloaded one per ten percent rather
// than one per chunk. ctx ends the pull; there is no other limit.
func PullModel(ctx context.Context, s config.ModelSettings, w io.Writer) error {
	client, err := newOllama(s)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "model-pull: pulling %s\n", s.Name)
	last, step := "", int64(-1)
	return client.Pull(ctx, s.Name, func(status string, completed, total int64) {
		if status != last {
			last, step = status, -1
			if total <= 0 {
				fmt.Fprintf(w, "model-pull: %s\n", status)
				return
			}
		}
		if total <= 0 || completed*10/total <= step {
			return
		}
		step = completed * 10 / total
		fmt.Fprintf(w, "model-pull: %s %d%%\n", status, step*10)
	})
}
