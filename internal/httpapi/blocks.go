package httpapi

import (
	"strings"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/richtext"
)

// Blocktypen des Editors. Nur "item" (Checkbox) zählt zum Fortschritt.
var blockKinds = []string{"item", "h1", "h2", "h3", "h4", "text", "bullet", "numbered", "quote", "code", "divider"}

// Sprachen für Codeblöcke ("" = automatisch/Plain).
var codeLangs = []string{"", "bash", "powershell", "sql", "json", "yaml", "javascript", "python", "text"}

const (
	maxRichLen = 20000 // bereinigtes HTML eines Titels/einer Notiz
	maxCodeLen = 50000 // Codeblock (Klartext)
)

// block sind die inhaltlichen Felder eines Checklisten- oder Template-Eintrags.
type block struct {
	Kind        string
	Title       string
	Description string
	NoteColor   string
	BgColor     string
	Indent      int64
	Lang        string
}

// normalize validiert und bereinigt einen Block. Titel und Notiz sind
// Inline-HTML (Whitelist), Codeblöcke speichern Klartext.
func (v validator) normalize(b *block) {
	v.oneOf("kind", b.Kind, blockKinds...)
	switch b.Kind {
	case "code":
		if !utf8.ValidString(b.Title) {
			v["title"] = "invalid_chars"
		} else if utf8.RuneCountInString(b.Title) > maxCodeLen {
			v["title"] = "too_long"
		}
	case "divider":
		b.Title = ""
	default:
		b.Title = v.rich("title", b.Title)
	}
	b.Description = v.rich("description", b.Description)
	v.oneOf("note_color", b.NoteColor, noteColors...)
	v.oneOf("bg_color", b.BgColor, bgColors...)
	if b.Indent < 0 || b.Indent > 4 {
		v["indent"] = "invalid_value"
	}
	v.oneOf("lang", b.Lang, codeLangs...)
}

func (v validator) rich(field, s string) string {
	if !utf8.ValidString(s) {
		v[field] = "invalid_chars"
		return s
	}
	s = richtext.Sanitize(s)
	if len(s) > maxRichLen {
		v[field] = "too_long"
	}
	return s
}

// status prüft den Freitext-Status eines Eintrags: Klartext, eine Zeile, max. 60 Zeichen.
func (v validator) status(s string) string {
	if strings.ContainsAny(s, "\r\n\t") {
		v["status"] = "invalid_chars"
		return ""
	}
	return v.text("status", s, 0, 60)
}
