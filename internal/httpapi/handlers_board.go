package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Board of the overview: order of the projects per column (shared by all
// users) and the personal customer filter (synced across devices).

const prefBoardHidden = "board_hidden"

// maxBoardIDs limits the size of order and filter lists.
const maxBoardIDs = 5000

// PUT /api/board/order {"status": "review", "ids": [..]}: the projects of a
// column in their new order. Moves projects into the column if necessary.
func (s *Server) orderBoard(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		Status string  `json:"status"`
		IDs    []int64 `json:"ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	v.oneOf("status", req.Status, projectStatuses...)
	if len(req.IDs) > maxBoardIDs {
		v["ids"] = "max:5000"
	}
	if err := v.err(); err != nil {
		return err
	}
	ctx := r.Context()
	for _, id := range req.IDs {
		if _, err := s.loadProject(ctx, u, id); err != nil {
			return err
		}
	}
	err := db.WithTx(ctx, s.db, func(q *sqlc.Queries) error {
		for i, id := range req.IDs {
			if err := q.SetProjectBoardPlace(ctx, sqlc.SetProjectBoardPlaceParams{Status: req.Status, BoardPosition: int64(i), ID: id}); err != nil {
				return err
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

// boardHidden returns the customers the user has hidden on the board.
func (s *Server) boardHidden(ctx context.Context, userID int64) ([]int64, error) {
	ids := []int64{}
	raw, err := s.q.GetUserPref(ctx, sqlc.GetUserPrefParams{UserID: userID, Key: prefBoardHidden})
	if errors.Is(err, sql.ErrNoRows) {
		return ids, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []int64{}, nil //nolint:nilerr // broken value: behave as "nothing hidden"
	}
	return ids, nil
}

// PUT /api/prefs/board-hidden {"customer_ids": [..]}: customers hidden on the board.
func (s *Server) setBoardHidden(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		CustomerIDs []int64 `json:"customer_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if len(req.CustomerIDs) > maxBoardIDs {
		return errBadRequest("too_many")
	}
	if req.CustomerIDs == nil {
		req.CustomerIDs = []int64{}
	}
	raw, err := json.Marshal(req.CustomerIDs)
	if err != nil {
		return err
	}
	if err := s.q.SetUserPref(r.Context(), sqlc.SetUserPrefParams{UserID: u.ID, Key: prefBoardHidden, Value: string(raw)}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Personal settings of the web app that follow the user across devices, stored
// as opaque JSON: the overview filters ("overview") and the saved views
// ("saved_views"), the own background ("background") and the display options of
// the page headers ("display", per page and view). Only these keys are accepted.
var clientPrefKeys = []string{"overview", "saved_views", "background", "display"}

const maxClientPref = 32 << 10

// clientPrefs returns the stored settings as key → JSON (missing keys are left out).
func (s *Server) clientPrefs(ctx context.Context, userID int64) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	for _, k := range clientPrefKeys {
		raw, err := s.q.GetUserPref(ctx, sqlc.GetUserPrefParams{UserID: userID, Key: k})
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[k] = json.RawMessage(raw)
	}
	return out, nil
}

// PUT /api/prefs/{key} (any JSON value, max. 32 KB).
func (s *Server) setClientPref(w http.ResponseWriter, r *http.Request, u authz.User) error {
	key := r.PathValue("key")
	if !slices.Contains(clientPrefKeys, key) {
		return errNotFound()
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxClientPref))
	if err != nil {
		return errBadRequest("too_large")
	}
	if !json.Valid(body) {
		return errBadRequest("invalid_json")
	}
	if err := s.q.SetUserPref(r.Context(), sqlc.SetUserPrefParams{UserID: u.ID, Key: key, Value: string(body)}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
