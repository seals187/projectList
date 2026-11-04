package httpapi

import (
	"database/sql"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// API response types. sqlc models are never returned directly so that e.g.
// password_hash never ends up in a response.

type userDTO struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	CustomerIDs []int64   `json:"customer_ids"`
}

func toUserDTO(u sqlc.User, customerIDs []int64) userDTO {
	if customerIDs == nil {
		customerIDs = []int64{}
	}
	return userDTO{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt, CustomerIDs: customerIDs}
}

const kindItem = "item"

var noteColors = []string{"", "green", "orange", "red", "indigo"}

// bgColors: background of a row or (on a heading) of the whole section.
// "" = same as section, "none" = explicitly no background. Three shades per
// hue as in Word/Excel: light, "-mid", "-strong". Must match web/src/lib/palette.ts;
// the database does not check the values (since migration 00006).
var bgColors = func() []string {
	out := []string{"", "none"}
	for _, shade := range []string{"", "-mid", "-strong"} {
		for _, hue := range []string{"gray", "yellow", "green", "blue", "red", "purple"} {
			out = append(out, hue+shade)
		}
	}
	return out
}()

type itemDTO struct {
	ID           int64      `json:"id"`
	ChecklistID  int64      `json:"checklist_id"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	NoteColor    string     `json:"note_color"`
	BgColor      string     `json:"bg_color"`
	Status       string     `json:"status"`
	Indent       int64      `json:"indent"`
	Lang         string     `json:"lang"`
	Position     int64      `json:"position"`
	AssigneeID   *int64     `json:"assignee_id"`
	AssigneeName *string    `json:"assignee_name"`
	DueDate      *string    `json:"due_date"`
	DoneAt       *time.Time `json:"done_at"`
	DoneBy       *int64     `json:"done_by"`
	DoneByName   *string    `json:"done_by_name"`
}

func nullStr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// toItemDTO accepts ListItemsByChecklistRow; the structurally identical type
// GetItemFullRow is passed via conversion.
func toItemDTO(r sqlc.ListItemsByChecklistRow) itemDTO {
	return itemDTO{
		ID: r.ID, ChecklistID: r.ChecklistID, Kind: r.Kind, Title: r.Title, Description: r.Description,
		NoteColor: r.NoteColor, BgColor: r.BgColor, Status: r.Status, Indent: r.Indent, Lang: r.Lang, Position: r.Position, AssigneeID: r.AssigneeID, AssigneeName: nullStr(r.AssigneeName),
		DueDate: r.DueDate, DoneAt: r.DoneAt, DoneBy: r.DoneBy, DoneByName: nullStr(r.DoneByName),
	}
}

// checklistDTO is a checklist incl. its entries.
type checklistDTO struct {
	ID           int64     `json:"id"`
	ProjectID    int64     `json:"project_id"`
	Title        string    `json:"title"`
	TemplateID   *int64    `json:"template_id"`
	Position     int64     `json:"position"`
	AssigneeID   *int64    `json:"assignee_id"`
	AssigneeName *string   `json:"assignee_name"`
	Total        int       `json:"total"`
	Done         int       `json:"done"`
	Items        []itemDTO `json:"items"`
}

func toChecklistDTO(c sqlc.Checklist, items []itemDTO) checklistDTO {
	d := checklistDTO{ID: c.ID, ProjectID: c.ProjectID, Title: c.Title, TemplateID: c.TemplateID, Position: c.Position, AssigneeID: c.AssigneeID, Items: items}
	if d.Items == nil {
		d.Items = []itemDTO{}
	}
	for _, it := range d.Items {
		if it.Kind != kindItem {
			continue
		}
		d.Total++
		if it.DoneAt != nil {
			d.Done++
		}
	}
	return d
}

type progress struct {
	Total int64 `json:"total"`
	Done  int64 `json:"done"`
}
