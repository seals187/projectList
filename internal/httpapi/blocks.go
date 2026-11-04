package httpapi

import (
	"strings"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/richtext"
)

// Block types of the editor. Only "item" (checkbox) counts towards progress.
var blockKinds = []string{"item", "h1", "h2", "h3", "h4", "text", "bullet", "numbered", "quote", "code", "divider"}

// Languages for code blocks ("" = automatic/plain).
var codeLangs = []string{"", "bash", "powershell", "sql", "json", "yaml", "javascript", "python", "text"}

const (
	maxRichLen = 20000 // sanitized HTML of a title/note
	maxCodeLen = 50000 // code block (plain text)
)

// block holds the content fields of a checklist or template entry.
type block struct {
	Kind        string
	Title       string
	Description string
	NoteColor   string
	BgColor     string
	Indent      int64
	Lang        string
}

// normalize validates and sanitizes a block. Title and note are
// inline HTML (whitelist), code blocks store plain text.
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

// status checks the free-text status of an entry: plain text, one line, max. 60 characters.
func (v validator) status(s string) string {
	if strings.ContainsAny(s, "\r\n\t") {
		v["status"] = "invalid_chars"
		return ""
	}
	return v.text("status", s, 0, 60)
}
