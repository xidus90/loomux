package cli

import (
	"testing"

	"github.com/xidus90/loomux/internal/flow/model"
)

func TestSchemaIsSortedAndClosed(t *testing.T) {
	got := Schema(map[string]model.ReplyType{"verdict": model.ReplyString, "n": model.ReplyInt, "ok": model.ReplyBool})
	want := `{"additionalProperties":false,"properties":{"n":{"type":"integer"},"ok":{"type":"boolean"},"verdict":{"type":"string"}},"required":["n","ok","verdict"],"type":"object"}`
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestFieldsAccepts(t *testing.T) {
	r := map[string]model.ReplyType{"s": model.ReplyString, "n": model.ReplyInt, "b": model.ReplyBool}
	got, err := Fields([]byte(`{"s":"a","n":3,"b":true}`), r)
	if err != nil || got["s"] != "a" || got["n"] != 3 || got["b"] != true {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestFieldsRejects(t *testing.T) {
	r := map[string]model.ReplyType{"n": model.ReplyInt}
	cases := map[string]string{
		`{"n":1,"x":2}`: `reply carries "x", which the node did not ask for`,
		`{}`:            `reply lacks "n"`,
		`{"n":"3"}`:     `reply field "n" is "3", not int`,
		`{"n":1.5}`:     `reply field "n" is 1.5, not int`,
		`{"n":null}`:    `reply field "n" is null, not int`,
	}
	for in, want := range cases {
		if _, err := Fields([]byte(in), r); err == nil || err.Error() != want {
			t.Errorf("%s: got %v want %s", in, err, want)
		}
	}
	if _, err := Fields([]byte(`nope`), r); err == nil {
		t.Error("garbage accepted")
	}
	if _, err := Fields([]byte(`{"s":1}`), map[string]model.ReplyType{"s": model.ReplyString}); err == nil {
		t.Error("string accepted a number")
	}
	if _, err := Fields([]byte(`{"b":1}`), map[string]model.ReplyType{"b": model.ReplyBool}); err == nil {
		t.Error("bool accepted a number")
	}
}

func TestFieldsRejectsNull(t *testing.T) {
	for _, tp := range []model.ReplyType{model.ReplyString, model.ReplyInt, model.ReplyBool} {
		if _, err := Fields([]byte(`{"f":null}`), map[string]model.ReplyType{"f": tp}); err == nil {
			t.Errorf("null accepted for %s", tp)
		}
	}
}
