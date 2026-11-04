package richtext

import "testing"

func TestSanitize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain & simple", "plain &amp; simple"},
		{"<b>fett</b> <strong>auch</strong>", "<b>fett</b> <b>auch</b>"},
		{"<em>k</em><del>x</del><strike>y</strike><u>u</u><code>c</code>", "<i>k</i><s>x</s><s>y</s><u>u</u><code>c</code>"},
		{`<span data-color="red">rot</span>`, `<span data-color="red">rot</span>`},
		{`<span data-color="hotpink" style="x">nein</span>`, `nein`},
		{`<span style="color:red">css</span>`, `css`},
		{`<mark data-bg="yellow">gelb</mark>`, `<mark data-bg="yellow">gelb</mark>`},
		{`<mark>default</mark>`, `<mark data-bg="yellow">default</mark>`},
		{`<a href="https://example.com" onclick="evil()">link</a>`, `<a href="https://example.com" target="_blank" rel="noopener noreferrer">link</a>`},
		{`<a href="javascript:alert(1)">x</a>`, `x`},
		{`<img src=x onerror=alert(1)>bild`, `bild`},
		{`<script>alert(1)</script>ok`, `ok`},
		{`<div>a</div><div>b</div>`, `a<br>b`},
		{`zeile<br>zwei`, `zeile<br>zwei`},
		{`<b><i>verschachtelt</i></b>`, `<b><i>verschachtelt</i></b>`},
		{`<font color="red">alt</font>`, `alt`},
		{`a < b`, `a &lt; b`},
	}
	for _, c := range cases {
		if got := Sanitize(c.in); got != c.want {
			t.Errorf("Sanitize(%q) = %q, erwartet %q", c.in, got, c.want)
		}
	}
}

func TestText(t *testing.T) {
	if got := Text(`<b>a</b> &amp; <span data-color="red">b</span>`); got != "a & b" {
		t.Fatalf("Text: %q", got)
	}
}

func TestPaletteColors(t *testing.T) {
	cases := map[string]string{
		// neue Stufen bleiben erhalten
		`<mark data-bg="yellow-strong">a</mark>`: `<mark data-bg="yellow-strong">a</mark>`,
		`<mark data-bg="gray-mid">a</mark>`:      `<mark data-bg="gray-mid">a</mark>`,
		`<span data-color="red-strong">a</span>`: `<span data-color="red-strong">a</span>`,
		`<span data-color="yellow-mid">a</span>`: `<span data-color="yellow-mid">a</span>`,
		`<span data-color="orange">a</span>`:     `<span data-color="orange">a</span>`, // ältere Einträge
		// unbekannte Werte: Textmarker fällt auf Gelb zurück, Textfarbe entfällt
		`<mark data-bg="pink-strong">a</mark>`:      `<mark data-bg="yellow">a</mark>`,
		`<span data-color="orange-strong">a</span>`: `a`,
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, erwartet %q", in, got, want)
		}
	}
}
