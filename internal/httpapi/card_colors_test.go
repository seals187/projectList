package httpapi_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCardColors(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c struct {
		ID int64 `json:"id"`
	}
	admin.do("POST", "/api/customers", map[string]any{"name": "A"}, &c)
	member := e.user("member@example.com", "member", c.ID)
	viewer := e.user("viewer@example.com", "viewer", c.ID)
	cu := fmt.Sprintf("/api/customers/%d/colors", c.ID)

	// shared customer color: partial updates, validation, writers only
	expect(t, "kundenfarbe", admin.do("PUT", cu, map[string]any{"color": "red-strong"}, nil), 200)
	expect(t, "überall", admin.do("PUT", cu, map[string]any{"everywhere": true}, nil), 200)
	expect(t, "ungültige farbe", admin.do("PUT", cu, map[string]any{"color": "#123456"}, nil), 400)
	expect(t, "ungültige stufe", admin.do("PUT", cu, map[string]any{"color": "red-dark"}, nil), 400)
	expect(t, "viewer darf nicht", viewer.do("PUT", cu, map[string]any{"color": "blue"}, nil), 403)

	type side struct {
		Customers []struct {
			Color    string `json:"color"`
			ColorAll bool   `json:"color_everywhere"`
		} `json:"customers"`
		Prefs map[string]json.RawMessage `json:"prefs"`
	}
	var s side
	member.do("GET", "/api/sidebar", nil, &s)
	if got := s.Customers[0]; got.Color != "red-strong" || !got.ColorAll {
		t.Fatalf("kundenfarbe für alle: %+v", got)
	}

	// personal card colors: a pref per user, not visible to others
	mine := map[string]any{"cards": map[string]string{fmt.Sprint(c.ID): "blue"}, "board": map[string]string{"7": "violet-strong"}}
	expect(t, "eigene farben", member.do("PUT", "/api/prefs/card_colors", mine, nil), 204)
	member.do("GET", "/api/sidebar", nil, &s)
	if string(s.Prefs["card_colors"]) == "" {
		t.Fatalf("eigene farben fehlen: %+v", s.Prefs)
	}
	var other side
	admin.do("GET", "/api/sidebar", nil, &other)
	if _, ok := other.Prefs["card_colors"]; ok {
		t.Fatalf("farben eines anderen sichtbar: %s", other.Prefs["card_colors"])
	}
}
