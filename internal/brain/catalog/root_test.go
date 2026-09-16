package catalog_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/config"
)

func TestRenderRootCatalog(t *testing.T) {
	out := catalog.RenderRootCatalog(nil)
	expectedEmpty := "# brain\n\n"
	if out != expectedEmpty {
		t.Errorf("expected %q, got %q", expectedEmpty, out)
	}

	areas := []config.Area{
		{Scope: "project/beta"},
		{Scope: "project/alpha"},
		{Scope: "core/hub"},
	}

	out = catalog.RenderRootCatalog(areas)
	expected := "# brain\n\n* [core/hub](brain://core/hub/)\n* [project/alpha](brain://project/alpha/)\n* [project/beta](brain://project/beta/)\n"
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}
