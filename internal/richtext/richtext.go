// Package richtext bereinigt die Inline-Formatierung der Blöcke (Titel und
// Notizen) per Whitelist. Erlaubt sind nur:
//
//	<b> <i> <u> <s> <code> <br>
//	<a href="http(s)://…|mailto:…">           (target/rel werden erzwungen)
//	<span data-color="gray|red|orange|green|blue|purple">
//	<mark data-bg="yellow|green|blue|red|purple">
//
// Synonyme (strong, em, strike, del) werden normalisiert. Alle anderen Tags
// werden entfernt, ihr Text bleibt erhalten; script/style samt Inhalt fallen
// komplett weg. Attribute außer den genannten werden verworfen.
package richtext

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Textfarben und Textmarker: 6 Farbtöne × 3 Stufen (hell, "-mid", "-strong"),
// gleich wie die Zeilenfarben (web/src/lib/palette.ts). "orange" bleibt als
// Textfarbe für ältere Einträge gültig.
var (
	TextColors      = palette("orange")
	HighlightColors = palette()
)

func palette(extra ...string) map[string]bool {
	m := map[string]bool{}
	for _, shade := range []string{"", "-mid", "-strong"} {
		for _, hue := range []string{"gray", "yellow", "green", "blue", "red", "purple"} {
			m[hue+shade] = true
		}
	}
	for _, e := range extra {
		m[e] = true
	}
	return m
}

var tagMap = map[string]string{
	"b": "b", "strong": "b",
	"i": "i", "em": "i",
	"u": "u", "ins": "u",
	"s": "s", "strike": "s", "del": "s",
	"code": "code",
	"a":    "a",
	"span": "span",
	"mark": "mark",
	"br":   "br",
}

// Sanitize gibt bereinigtes HTML zurück.
func Sanitize(in string) string {
	if !strings.ContainsAny(in, "<&") {
		return html.EscapeString(in)
	}
	ctx := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(in), ctx)
	if err != nil {
		return html.EscapeString(in)
	}
	var b strings.Builder
	for i, n := range nodes {
		// Top-Level-Blöcke (z. B. <div>-Zeilen aus contenteditable) als Umbruch erhalten.
		if i > 0 && n.Type == html.ElementNode && isBlock(n.DataAtom) {
			b.WriteString("<br>")
		}
		render(&b, n, 0)
	}
	return strings.TrimSpace(b.String())
}

// Text liefert den reinen Text (z. B. für Längenprüfung oder Suche).
func Text(in string) string {
	nodes, err := html.ParseFragment(strings.NewReader(in), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return in
	}
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return b.String()
}

func render(b *strings.Builder, n *html.Node, depth int) {
	if depth > 20 {
		return
	}
	switch n.Type {
	case html.TextNode:
		b.WriteString(html.EscapeString(n.Data))
		return
	case html.ElementNode:
	default:
		return
	}

	if n.DataAtom == atom.Script || n.DataAtom == atom.Style || n.DataAtom == atom.Template {
		return
	}
	tag, ok := tagMap[n.Data]
	open := ""
	if ok {
		switch tag {
		case "br":
			b.WriteString("<br>")
			return
		case "a":
			if href := attr(n, "href"); safeURL(href) {
				open = `<a href="` + html.EscapeString(href) + `" target="_blank" rel="noopener noreferrer">`
			}
		case "span":
			if c := attr(n, "data-color"); TextColors[c] {
				open = `<span data-color="` + c + `">`
			}
		case "mark":
			c := attr(n, "data-bg")
			if !HighlightColors[c] {
				c = "yellow"
			}
			open = `<mark data-bg="` + c + `">`
		default:
			open = "<" + tag + ">"
		}
	}
	// Block-Elemente (div, p, li …) als Zeilenumbruch erhalten.
	if !ok && isBlock(n.DataAtom) && n.PrevSibling != nil {
		b.WriteString("<br>")
	}
	b.WriteString(open)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		render(b, c, depth+1)
	}
	if open != "" {
		b.WriteString("</" + tag + ">")
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func safeURL(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "mailto:")
}

func isBlock(a atom.Atom) bool {
	switch a {
	case atom.Div, atom.P, atom.Li, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Blockquote, atom.Pre:
		return true
	}
	return false
}
