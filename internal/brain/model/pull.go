package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrUnreachable marks an error of Has that got no answer at all: Ollama is
// not running there, or not in time.
var ErrUnreachable = errors.New("ollama cannot be reached")

// Has says whether Ollama already holds the model name; names are compared
// in any case, as Ollama compares them. The error stands for an Ollama that
// cannot be asked: no connection (ErrUnreachable), a status that is not 2xx,
// an answer that is not the list of models.
func (c *Client) Has(ctx context.Context, name string) (bool, error) {
	// The URL passed the guard, so the request always builds.
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/tags", nil)
	response, err := c.http.Do(request)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return false, fmt.Errorf("GET /api/tags answered %s", response.Status)
	}
	var list struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		return false, fmt.Errorf("GET /api/tags answered unreadably: %w", err)
	}
	want := tagged(name)
	for _, m := range list.Models {
		if strings.EqualFold(tagged(m.Name), want) || strings.EqualFold(tagged(m.Model), want) {
			return true, nil
		}
	}
	return false, nil
}

// tagged is name with the tag Ollama gives a name without one. The tag
// follows a colon in the last path segment; a colon before a slash belongs
// to a registry's port.
func tagged(name string) string {
	if strings.Contains(name[strings.LastIndex(name, "/")+1:], ":") {
		return name
	}
	return name + ":latest"
}

// pullLine is one line of Ollama's streamed answer to a pull.
type pullLine struct {
	Status    string  `json:"status"`
	Error     *string `json:"error"`
	Completed int64   `json:"completed"`
	Total     int64   `json:"total"`
}

// Pull downloads the model name into Ollama and hands every line of the
// stream to progress. Only `loomux init` calls it, and only on a human's
// confirmation; reconcile never pulls, it asks the model that is there or
// counts the case as manual.
//
// There is no total limit: the download ends with Ollama's success, with an
// error, or with ctx. An error line, a status that is not 2xx, a line that is
// not JSON and a stream that ends without success are failures.
func (c *Client) Pull(ctx context.Context, name string, progress func(status string, completed, total int64)) error {
	// A struct of a string and a bool always marshals; the URL passed the guard.
	body, _ := json.Marshal(struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}{name, true})
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/pull", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := c.pull.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		// Ollama says why in {"error": …}; a body without one says nothing.
		var answer struct{ Error string }
		if json.NewDecoder(response.Body).Decode(&answer) == nil && answer.Error != "" {
			return fmt.Errorf("POST /api/pull answered %s: %s", response.Status, answer.Error)
		}
		return fmt.Errorf("POST /api/pull answered %s", response.Status)
	}
	lines := bufio.NewScanner(response.Body)
	lines.Buffer(make([]byte, 64*1024), 1024*1024)
	for lines.Scan() {
		text := bytes.TrimSpace(lines.Bytes())
		if len(text) == 0 {
			continue
		}
		var line pullLine
		if err := json.Unmarshal(text, &line); err != nil {
			return fmt.Errorf("ollama answered unreadably: %s", text)
		}
		if line.Error != nil {
			if *line.Error == "" {
				return errors.New("ollama pull failed")
			}
			return errors.New(*line.Error)
		}
		progress(line.Status, line.Completed, line.Total)
		if line.Status == "success" {
			return nil
		}
	}
	if err := lines.Err(); err != nil {
		return err
	}
	return errors.New("the pull ended without success")
}
