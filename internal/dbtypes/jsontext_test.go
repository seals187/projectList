package dbtypes

import (
	"encoding/json"
	"testing"
)

func TestJSONText(t *testing.T) {
	for _, src := range []any{`[{"number":"1"}]`, []byte(`[{"number":"1"}]`)} {
		var j JSONText
		if err := j.Scan(src); err != nil || string(j) != `[{"number":"1"}]` {
			t.Fatalf("scan %T: %v %q", src, err, j)
		}
	}
	var n JSONText
	if err := n.Scan(nil); err != nil || string(n) != "[]" {
		t.Fatalf("scan nil: %v %q", err, n)
	}
	if v, _ := JSONText(nil).Value(); v != "[]" {
		t.Fatalf("value empty: %v", v)
	}
	if v, _ := JSONText(`[1]`).Value(); v != "[1]" {
		t.Fatalf("value: %v", v)
	}
	b, _ := json.Marshal(struct {
		P JSONText `json:"p"`
	}{JSONText(`[{"a":1}]`)})
	if string(b) != `{"p":[{"a":1}]}` {
		t.Fatalf("marshal: %s", b)
	}
}
