package service

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Domain events of the project lifecycle (they are the automation trigger kinds).
const (
	EventProjectCreated = "project_created" // fields: name, customer, status
	EventProjectTag     = "project_tag"     // a tag was added; extra field: tag
	EventProjectStatus  = "project_status"  // the status changed
)

// Activity kinds of the project activity log (also used by the web app for the texts).
const (
	ActProjectCreated   = "project_created"
	ActRenamed          = "renamed"
	ActStatus           = "status"
	ActDates            = "dates"
	ActOwner            = "owner"
	ActTags             = "tags"
	ActRedmine          = "redmine"
	ActChecklistCreated = "checklist_created"
	ActChecklistDone    = "checklist_completed"
	ActChecklistDeleted = "checklist_deleted"
	ActItemDone         = "item_done"
	ActFileAdded        = "file_added"
	ActFileDeleted      = "file_deleted"
	ActTableCreated     = "table_created"
	ActTableDeleted     = "table_deleted"
)

// Actor is who causes a change: a user, or an automation (ID nil).
type Actor struct {
	ID   *int64
	Name string // used for the variable {{user}}
}

// UserActor returns the actor for a logged-in user.
func UserActor(id int64, name string) Actor { return Actor{ID: &id, Name: name} }

// Automation is the actor of automation rules.
var Automation = Actor{Name: "Automation"}

func (a Actor) automated() bool { return a.ID == nil }

// Lifecycle owns the changes of a project that have side effects: every change
// of the status, name, dates, owner or tags is written to the activity log and
// emitted as a domain event (automation), no matter which endpoint caused it.
// The side effects run after the change is committed; their failures are only
// logged.
type Lifecycle struct {
	DB  *sql.DB
	Q   *sqlc.Queries
	Log *slog.Logger
	Now func() time.Time // nil = time.Now
	// Emit receives the domain events (kind, project, extra fields);
	// nil = no events. Production: the automation engine.
	Emit func(ctx context.Context, kind string, p sqlc.Project, extra map[string]string)
}

func (l *Lifecycle) now() time.Time {
	if l.Now == nil {
		return time.Now()
	}
	return l.Now()
}

func (l *Lifecycle) emit(ctx context.Context, kind string, p sqlc.Project, extra map[string]string) {
	if l.Emit != nil {
		l.Emit(ctx, kind, p, extra)
	}
}

// LogActivity records an entry in the project's activity log. A failure is
// only logged: the change itself has already been saved.
func (l *Lifecycle) LogActivity(ctx context.Context, projectID int64, a Actor, kind, subject, detail string) {
	if a.automated() && detail == "" {
		detail = "automation"
	}
	err := l.Q.AddProjectActivity(ctx, sqlc.AddProjectActivityParams{
		ProjectID: projectID, UserID: a.ID, Kind: kind, Subject: subject, Detail: detail, CreatedAt: l.now().UTC(),
	})
	if err != nil && l.Log != nil {
		l.Log.Warn("aktivität nicht gespeichert", "project", projectID, "kind", kind, "err", err)
	}
}

// CreateProject creates a project (status from p, normally "planning").
func (l *Lifecycle) CreateProject(ctx context.Context, a Actor, p sqlc.CreateProjectParams) (sqlc.Project, error) {
	pr, err := l.Q.CreateProject(ctx, p)
	if err != nil {
		return pr, err
	}
	l.LogActivity(ctx, pr.ID, a, ActProjectCreated, "", "")
	l.emit(ctx, EventProjectCreated, pr, nil)
	return pr, nil
}

// UpdateProject saves the new properties of prev. A new owner takes over the
// checklists without a person and those of the previous owner.
func (l *Lifecycle) UpdateProject(ctx context.Context, a Actor, prev sqlc.Project, up sqlc.UpdateProjectParams) (sqlc.Project, error) {
	up.ID = prev.ID
	p, err := l.Q.UpdateProject(ctx, up)
	if err != nil {
		return p, err
	}
	if !samePtr(p.OwnerID, prev.OwnerID) {
		if err := l.Q.AssignChecklistsToOwner(ctx, sqlc.AssignChecklistsToOwnerParams{NewOwner: p.OwnerID, ProjectID: p.ID, OldOwner: prev.OwnerID}); err != nil {
			return p, err
		}
	}
	l.changed(ctx, a, prev, p)
	return p, nil
}

// MoveOnBoard puts the projects (in this order) into the board column status.
// Projects that change their column change their status, with the same side
// effects as UpdateProject. Unknown ids: sql.ErrNoRows, nothing changes.
func (l *Lifecycle) MoveOnBoard(ctx context.Context, a Actor, status string, ids []int64) error {
	var moved []struct{ prev, now sqlc.Project }
	err := db.WithTx(ctx, l.DB, func(q *sqlc.Queries) error {
		moved = moved[:0]
		for i, id := range ids {
			prev, err := q.GetProject(ctx, id)
			if err != nil {
				return err
			}
			if err := q.SetProjectBoardPlace(ctx, sqlc.SetProjectBoardPlaceParams{Status: status, BoardPosition: int64(i), ID: id}); err != nil {
				return err
			}
			if prev.Status != status {
				now := prev
				now.Status = status
				moved = append(moved, struct{ prev, now sqlc.Project }{prev, now})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, m := range moved {
		l.changed(ctx, a, m.prev, m.now)
	}
	return nil
}

// changed writes the changed properties into the activity log and emits the
// status event.
func (l *Lifecycle) changed(ctx context.Context, a Actor, prev, p sqlc.Project) {
	if p.Name != prev.Name {
		l.LogActivity(ctx, p.ID, a, ActRenamed, prev.Name, p.Name)
	}
	if p.Status != prev.Status {
		l.LogActivity(ctx, p.ID, a, ActStatus, prev.Status, p.Status)
	}
	if deref(p.StartDate) != deref(prev.StartDate) || deref(p.DueDate) != deref(prev.DueDate) {
		l.LogActivity(ctx, p.ID, a, ActDates, deref(p.StartDate), deref(p.DueDate))
	}
	if !samePtr(p.OwnerID, prev.OwnerID) {
		name := ""
		if p.OwnerID != nil {
			if o, err := l.Q.GetUserByID(ctx, *p.OwnerID); err == nil {
				name = o.Name
			}
		}
		l.LogActivity(ctx, p.ID, a, ActOwner, "", name)
	}
	if p.RedmineTicket != prev.RedmineTicket {
		l.LogActivity(ctx, p.ID, a, ActRedmine, "", p.RedmineTicket)
	}
	if p.Status != prev.Status {
		l.emit(ctx, EventProjectStatus, p, nil)
	}
}

// SetTags replaces the tags of the project. tagNames are the names of tagIDs
// for the activity log. Returns the tags now set.
func (l *Lifecycle) SetTags(ctx context.Context, a Actor, projectID int64, tagIDs []int64, tagNames []string) ([]int64, error) {
	prev, err := l.Q.ListTagIDsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	err = db.WithTx(ctx, l.DB, func(q *sqlc.Queries) error {
		if err := q.ClearProjectTags(ctx, projectID); err != nil {
			return err
		}
		for _, t := range tagIDs {
			if err := q.AddProjectTag(ctx, sqlc.AddProjectTagParams{ProjectID: projectID, TagID: t}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	ids, err := l.Q.ListTagIDsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if slices.Equal(prev, ids) {
		return ids, nil
	}
	l.LogActivity(ctx, projectID, a, ActTags, "", strings.Join(tagNames, ", "))
	if p, err := l.Q.GetProject(ctx, projectID); err == nil {
		for _, t := range ids {
			if !slices.Contains(prev, t) {
				l.emit(ctx, EventProjectTag, p, map[string]string{"tag": strconv.FormatInt(t, 10)})
			}
		}
	}
	return ids, nil
}

// Instance is what Instantiate created: a checklist or a table.
type Instance struct {
	Checklist *sqlc.Checklist
	Table     *sqlc.ProjectTable
}

// InstantiateReq describes applying a template to a project.
type InstantiateReq struct {
	ProjectID  int64
	TemplateID int64
	Kind       string            // KindChecklist, KindTable or "" (whatever the template is)
	Title      string            // "" = template name
	Values     map[string]string // custom variable values ({{key}})
	RuleID     *int64            // automation rule that applies the template
}

// Instantiate creates a checklist or a table in the project as a snapshot of
// the template, with the variables resolved (system values, the actor's name,
// the given values), and logs it in the activity log. A template of another
// kind than req.Kind: ErrTemplateKind.
func (l *Lifecycle) Instantiate(ctx context.Context, a Actor, req InstantiateReq) (Instance, error) {
	now := l.now()
	tmpl, err := l.Q.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return Instance{}, err
	}
	if req.Kind != "" && tmpl.Kind != req.Kind {
		return Instance{}, ErrTemplateKind
	}
	vals, err := ResolveValues(ctx, l.Q, req.ProjectID, a.Name, req.Values, now)
	if err != nil {
		return Instance{}, err
	}
	opts := ApplyOptions{Values: vals, RuleID: req.RuleID, Now: now}
	if tmpl.Kind == KindTable {
		t, err := ApplyTableTemplate(ctx, l.DB, req.ProjectID, req.TemplateID, req.Title, a.ID, opts)
		if err != nil {
			return Instance{}, err
		}
		l.LogActivity(ctx, req.ProjectID, a, ActTableCreated, t.Title, "")
		return Instance{Table: &t}, nil
	}
	cl, err := ApplyTemplate(ctx, l.DB, req.ProjectID, req.TemplateID, req.Title, opts)
	if err != nil {
		return Instance{}, err
	}
	l.LogActivity(ctx, req.ProjectID, a, ActChecklistCreated, cl.Title, "")
	return Instance{Checklist: &cl}, nil
}

// Duplicate copies the project into the same customer under the given name:
// description, owner, tags, all checklists (entries not checked off, without
// due dates) and all tables. Status: planning.
func (l *Lifecycle) Duplicate(ctx context.Context, a Actor, src sqlc.Project, name string) (sqlc.Project, error) {
	var p sqlc.Project
	stamp := l.now().UTC().Format(time.RFC3339)
	err := db.WithTx(ctx, l.DB, func(q *sqlc.Queries) error {
		var err error
		p, err = q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: src.CustomerID, Name: name, Description: src.Description, Status: "planning"})
		if err != nil {
			return err
		}
		p, err = q.UpdateProject(ctx, sqlc.UpdateProjectParams{
			ID: p.ID, Name: p.Name, Description: p.Description, Status: p.Status, IncludeInCustomerProgress: src.IncludeInCustomerProgress,
			OwnerID: src.OwnerID, RedmineTicket: "",
		})
		if err != nil {
			return err
		}
		tagIDs, err := q.ListTagIDsByProject(ctx, src.ID)
		if err != nil {
			return err
		}
		for _, t := range tagIDs {
			if err := q.AddProjectTag(ctx, sqlc.AddProjectTagParams{ProjectID: p.ID, TagID: t}); err != nil {
				return err
			}
		}
		checklists, err := q.ListChecklistsByProject(ctx, src.ID)
		if err != nil {
			return err
		}
		for _, c := range checklists {
			cl, err := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: c.Title, TemplateID: c.TemplateID, Position: c.Position})
			if err != nil {
				return err
			}
			items, err := q.ListItemsByChecklist(ctx, c.ID)
			if err != nil {
				return err
			}
			for _, it := range items {
				ip := BlockOfItem(it).Item(cl.ID, it.Position)
				ip.AssigneeID = it.AssigneeID
				if _, err := q.CreateItem(ctx, ip); err != nil {
					return err
				}
			}
		}
		tables, err := q.ListTableDataByProject(ctx, src.ID)
		if err != nil {
			return err
		}
		for _, t := range tables {
			if _, err := q.CreateTable(ctx, sqlc.CreateTableParams{
				ProjectID: p.ID, Title: t.Title, TemplateID: t.TemplateID, Position: t.Position, Data: t.Data, CreatedAt: stamp, UpdatedAt: stamp, UpdatedBy: a.ID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return p, err
	}
	l.LogActivity(ctx, p.ID, a, ActProjectCreated, src.Name, "copy")
	l.emit(ctx, EventProjectCreated, p, nil)
	return p, nil
}

func samePtr[T comparable](a, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}
