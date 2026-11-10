package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
)

// Tables (spreadsheets) of a project. The workbook (sheets, cells, formulas,
// formatting) is edited and calculated in the web app (Fortune-sheet) and stored
// here as a JSON document; the server only checks its shape and size. Saving
// uses optimistic locking: the client sends the version it has loaded, an older
// version is rejected with 409 "conflict".

const (
	maxTableData   = 8 << 20 // bytes of the JSON document
	maxTableSheets = 50
)

// tableData validates a workbook: a JSON array of 1…50 sheet objects with a
// name; returned compacted. An empty document becomes one empty sheet.
func tableData(v validator, field string, raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return service.EmptyWorkbook
	}
	if len(raw) > maxTableData {
		v[field] = "too_large"
		return ""
	}
	var sheets []struct {
		Name *string `json:"name"`
	}
	if err := json.Unmarshal(raw, &sheets); err != nil || len(sheets) == 0 || len(sheets) > maxTableSheets {
		v[field] = "invalid_value"
		return ""
	}
	for _, sh := range sheets {
		if sh.Name == nil || *sh.Name == "" {
			v[field] = "invalid_value"
			return ""
		}
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		v[field] = "invalid_value"
		return ""
	}
	return buf.String()
}

// decodeLarge is decode with room for a workbook.
func decodeLarge(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxTableData+1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return errBadRequest("too_large")
		}
		return errBadRequest("invalid_request")
	}
	return nil
}

// stamp: the current time as stored in project_tables (RFC 3339, UTC).
func (s *Server) stamp() string { return s.now().UTC().Format(time.RFC3339) }

// tableDTO: a table with its workbook (data as JSON, not as a string).
type tableDTO struct {
	ID            int64           `json:"id"`
	ProjectID     int64           `json:"project_id"`
	Title         string          `json:"title"`
	TemplateID    *int64          `json:"template_id"`
	Position      int64           `json:"position"`
	Version       int64           `json:"version"`
	UpdatedAt     string          `json:"updated_at"`
	UpdatedByName string          `json:"updated_by_name"`
	Data          json.RawMessage `json:"data"`
}

func toTableDTO(t sqlc.ProjectTable, updatedBy string) tableDTO {
	return tableDTO{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, TemplateID: t.TemplateID, Position: t.Position,
		Version: t.Version, UpdatedAt: t.UpdatedAt, UpdatedByName: updatedBy, Data: json.RawMessage(t.Data),
	}
}

// loadTable loads a table and checks access via the customer.
func (s *Server) loadTable(ctx context.Context, u authz.User, id int64) (sqlc.GetTableWithCustomerRow, error) {
	t, err := s.q.GetTableWithCustomer(ctx, id)
	if err != nil {
		return t, err
	}
	return t, s.requireCustomer(ctx, u, t.CustomerID)
}

// GET /api/projects/{id}/tables
func (s *Server) listTables(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	ts, err := s.q.ListTablesByProject(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ts)
	return nil
}

// POST /api/projects/{id}/tables {title, data?}: empty table or imported workbook (XLSX import in the web app).
func (s *Server) createTable(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return err
	}
	var req struct {
		Title string          `json:"title"`
		Data  json.RawMessage `json:"data"`
	}
	if err := decodeLarge(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	title := v.text("title", req.Title, 1, 200)
	data := tableData(v, "data", req.Data)
	if err := v.err(); err != nil {
		return err
	}
	var t sqlc.ProjectTable
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		pos, err := q.NextTablePosition(r.Context(), projectID)
		if err != nil {
			return err
		}
		now := s.stamp()
		t, err = q.CreateTable(r.Context(), sqlc.CreateTableParams{
			ProjectID: projectID, Title: title, Position: pos, Data: data, CreatedAt: now, UpdatedAt: now, UpdatedBy: &u.ID,
		})
		return err
	})
	if err != nil {
		return err
	}
	s.logActivity(r.Context(), projectID, u.ID, actTableCreated, t.Title, "")
	writeJSON(w, http.StatusCreated, toTableDTO(t, u.Name))
	return nil
}

// POST /api/projects/{id}/tables/from-template {template_id, title, values}
func (s *Server) createTableFromTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return err
	}
	var req struct {
		TemplateID int64             `json:"template_id"`
		Title      string            `json:"title"`
		Values     map[string]string `json:"values"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	title := v.text("title", req.Title, 0, 200)
	if req.TemplateID <= 0 {
		v["template_id"] = "required"
	}
	if err := v.err(); err != nil {
		return err
	}
	if len(req.Values) > 100 {
		return errBadRequest("too_many_values")
	}
	vals, err := service.ResolveValues(r.Context(), s.q, projectID, u.Name, req.Values, s.now())
	if err != nil {
		return err
	}
	t, err := service.ApplyTableTemplate(r.Context(), s.db, projectID, req.TemplateID, title, &u.ID, service.ApplyOptions{Values: vals, Now: s.now()})
	if err != nil {
		return err
	}
	s.logActivity(r.Context(), projectID, u.ID, actTableCreated, t.Title, "")
	writeJSON(w, http.StatusCreated, toTableDTO(t, u.Name))
	return nil
}

// PUT /api/projects/{id}/tables/order {ids}
func (s *Server) orderTables(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return err
	}
	var req orderReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		for i, id := range req.IDs {
			n, err := q.SetTablePosition(r.Context(), sqlc.SetTablePositionParams{Position: int64(i), ID: id, ProjectID: projectID})
			if err != nil {
				return err
			}
			if n != 1 {
				return errBadRequest("invalid_order")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GET /api/tables/{id}: table with workbook, project and customer (header, breadcrumb).
func (s *Server) getTable(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	t, err := s.loadTable(r.Context(), u, id)
	if err != nil {
		return err
	}
	p, err := s.q.GetProject(r.Context(), t.ProjectID)
	if err != nil {
		return err
	}
	customer, err := s.q.GetCustomer(r.Context(), t.CustomerID)
	if err != nil {
		return err
	}
	pt := sqlc.ProjectTable{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, TemplateID: t.TemplateID, Position: t.Position,
		Data: t.Data, Version: t.Version, UpdatedAt: t.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"table":    toTableDTO(pt, t.UpdatedByName),
		"project":  map[string]any{"id": p.ID, "name": p.Name, "status": p.Status},
		"customer": map[string]any{"id": customer.ID, "name": customer.Name},
	})
	return nil
}

// PATCH /api/tables/{id} {title}
func (s *Server) updateTable(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadTable(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	title := v.text("title", req.Title, 1, 200)
	if err := v.err(); err != nil {
		return err
	}
	t, err := s.q.UpdateTableTitle(r.Context(), sqlc.UpdateTableTitleParams{Title: title, ID: id})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": t.ID, "title": t.Title})
	return nil
}

// PUT /api/tables/{id}/data {data, version} → {version, updated_at}; 409 "conflict"
// when someone else saved in between (the client reloads).
func (s *Server) saveTableData(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadTable(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Data    json.RawMessage `json:"data"`
		Version int64           `json:"version"`
	}
	if err := decodeLarge(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	data := tableData(v, "data", req.Data)
	if err := v.err(); err != nil {
		return err
	}
	res, err := s.q.SaveTableData(r.Context(), sqlc.SaveTableDataParams{
		Data: data, UpdatedAt: s.stamp(), UpdatedBy: &u.ID, ID: id, Version: req.Version,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return errConflict("conflict")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"version": res.Version, "updated_at": res.UpdatedAt, "updated_by_name": u.Name})
	return nil
}

// DELETE /api/tables/{id}
func (s *Server) deleteTable(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	t, err := s.loadTable(r.Context(), u, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteTable(r.Context(), id); err != nil {
		return err
	}
	s.logActivity(r.Context(), t.ProjectID, u.ID, actTableDeleted, t.Title, "")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// POST /api/tables/{id}/save-as-template {name, description}: new table template with the workbook.
func (s *Server) saveTableAsTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	t, err := s.loadTable(r.Context(), u, id)
	if err != nil {
		return err
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	name := v.text("name", req.Name, 1, 200)
	desc := v.text("description", req.Description, 0, 10000)
	if err := v.err(); err != nil {
		return err
	}
	tmpl, err := s.q.CreateTableTemplate(r.Context(), sqlc.CreateTableTemplateParams{Name: name, Description: desc, SheetData: t.Data})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, tmpl)
	return nil
}

// PUT /api/templates/{id}/sheet {data}: workbook of a table template.
func (s *Server) saveTemplateSheet(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Data json.RawMessage `json:"data"`
	}
	if err := decodeLarge(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	data := tableData(v, "data", req.Data)
	if err := v.err(); err != nil {
		return err
	}
	n, err := s.q.UpdateTemplateSheet(r.Context(), sqlc.UpdateTemplateSheetParams{SheetData: data, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.q.GetTemplate(r.Context(), id); err != nil {
			return err // not found
		}
		return service.ErrTemplateKind
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
