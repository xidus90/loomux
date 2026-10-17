package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hosts"
)

// configLLM shows the llm_config switch of the area rooted at the project, or
// turns it with on or off. Only --root is taken: the switch belongs to one
// registered area, never to the global file.
func configLLM(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux config llm", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "project root; found upwards when empty")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) > 1 || (len(positional) == 1 && positional[0] != "on" && positional[0] != "off") {
		return configUsage(stderr)
	}
	if *root == "" {
		if *root, err = hosts.FindRoot("."); err != nil {
			fmt.Fprintf(stderr, "loomux config: %v\n", err)
			return 1
		}
	}
	state := config.StateDir()
	scope, on, err := guard.AreaAt(*root, state)
	// No registry yet is no area, the same as an area elsewhere.
	if errors.Is(err, os.ErrNotExist) {
		scope, err = "", nil
	}
	if err == nil && scope == "" {
		err = fmt.Errorf("%s is not a registered area; run `loomux init` first", *root)
	}
	if err == nil && len(positional) == 1 {
		on = positional[0] == "on"
		err = config.SetLLMConfig(state, scope, on)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	word := "off"
	if on {
		word = "on"
	}
	fmt.Fprintf(stdout, "llm_config: %s (%s)\n", word, scope)
	return 0
}

// llmConfigOn is the switch of the area rooted at root; anything unreadable
// counts as off, since the line it gates only announces an opening.
func llmConfigOn(root string) bool {
	_, on, err := guard.AreaAt(root, config.StateDir())
	return err == nil && on
}
