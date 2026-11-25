package service

import "github.com/seals187/projectList/internal/db/sqlc"

// Block is the content of one checklist entry or template entry: what is
// copied when a template is applied, a checklist is saved as a template or a
// project is duplicated (no status, done marks or due dates).
type Block struct {
	Kind        string
	Title       string
	Description string
	NoteColor   string
	BgColor     string
	Indent      int64
	Lang        string
}

// BlockOfItem returns the content of a checklist entry.
func BlockOfItem(it sqlc.ListItemsByChecklistRow) Block {
	return Block{Kind: it.Kind, Title: it.Title, Description: it.Description, NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang}
}

// BlockOfTemplateItem returns the content of a template entry.
func BlockOfTemplateItem(it sqlc.TemplateItem) Block {
	return Block{Kind: it.Kind, Title: it.Title, Description: it.Description, NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang}
}

// Item returns the parameters for creating the block as a checklist entry.
func (b Block) Item(checklistID, position int64) sqlc.CreateItemParams {
	return sqlc.CreateItemParams{
		ChecklistID: checklistID, Kind: b.Kind, Title: b.Title, Description: b.Description,
		NoteColor: b.NoteColor, BgColor: b.BgColor, Indent: b.Indent, Lang: b.Lang, Position: position,
	}
}

// TemplateItem returns the parameters for creating the block as a template entry.
func (b Block) TemplateItem(templateID, position int64) sqlc.CreateTemplateItemParams {
	return sqlc.CreateTemplateItemParams{
		TemplateID: templateID, Kind: b.Kind, Title: b.Title, Description: b.Description,
		NoteColor: b.NoteColor, BgColor: b.BgColor, Indent: b.Indent, Lang: b.Lang, Position: position,
	}
}
