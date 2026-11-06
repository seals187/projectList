package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/seals187/projectList/internal/db"
)

type restoreResult struct {
	Error      string `json:"error"`
	Version    int64  `json:"version"`
	SafetyCopy string `json:"safety_copy"`
	Relogin    bool   `json:"relogin"`
	CSRFToken  string `json:"csrf_token"`
}

// restoreUpload posts a backup to /api/admin/restore and decodes the answer.
func (c *client) restoreUpload(content []byte) (int, restoreResult) {
	c.e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "backup.db")
	fw.Write(content)
	mw.Close()
	req, _ := http.NewRequest("POST", c.e.srv.URL+"/api/admin/restore", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-CSRF-Token", c.csrf)
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out restoreResult
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

// tamper writes the backup to a file, runs SQL on it and returns the new bytes.
func tamper(t *testing.T, backup []byte, stmts ...string) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.db")
	if err := os.WriteFile(path, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	// back to a single file (no WAL left over)
	if _, err := d.Exec("PRAGMA journal_mode=DELETE"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRestore(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, c idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "Vorher"}, &a)
	member := e.user("member@example.com", "member", a.ID)

	backup := []byte(admin.raw("GET", "/api/admin/backup"))
	if !bytes.HasPrefix(backup, []byte("SQLite format 3")) {
		t.Fatalf("backup ist keine sqlite-datei (%d bytes)", len(backup))
	}

	// changes after the backup
	admin.do("POST", "/api/customers", map[string]string{"name": "Nachher"}, &c)

	// not allowed / invalid files are rejected before anything changes
	if st, _ := member.restoreUpload(backup); st != 403 {
		t.Fatalf("member darf nicht wiederherstellen: %d", st)
	}
	for name, tc := range map[string]struct {
		content []byte
		code    string
	}{
		"keine sqlite-datei":  {[]byte("hello world, no database"), "restore_not_sqlite"},
		"fremde datenbank":    {tamper(t, backup, "DROP TABLE goose_db_version"), "restore_not_projectlist"},
		"neuere version":      {tamper(t, backup, "INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999, 1)"), "restore_newer"},
		"ohne admin":          {tamper(t, backup, "UPDATE users SET role = 'member'"), "restore_no_admin"},
		"abgeschnittene datei": {backup[:len(backup)/2], ""},
	} {
		st, res := admin.restoreUpload(tc.content)
		if st != 400 || (tc.code != "" && res.Error != tc.code) {
			t.Fatalf("%s: status %d, fehler %q (erwartet 400 %q)", name, st, res.Error, tc.code)
		}
	}
	var cs []struct{ Name string }
	admin.do("GET", "/api/customers", nil, &cs)
	if len(cs) != 2 {
		t.Fatalf("abgelehnte wiederherstellung hat daten verändert: %+v", cs)
	}

	// restore: data as at the time of the backup, admin stays signed in
	st, res := admin.restoreUpload(backup)
	if st != 200 || res.Relogin || res.CSRFToken == "" || res.SafetyCopy == "" {
		t.Fatalf("wiederherstellen: %d %+v", st, res)
	}
	admin.csrf = res.CSRFToken
	cs = nil
	expect(t, "kunden nach restore", admin.do("GET", "/api/customers", nil, &cs), 200)
	if len(cs) != 1 || cs[0].Name != "Vorher" {
		t.Fatalf("daten nach restore: %+v", cs)
	}
	// writes work after the restore (new CSRF token)
	expect(t, "schreiben nach restore", admin.do("POST", "/api/customers", map[string]string{"name": "Neu"}, nil), 201)

	// the safety copy of the state before the restore is a valid backup with "Nachher"
	copyPath := filepath.Join(e.dir, res.SafetyCopy)
	if _, err := db.CheckBackup(t.Context(), copyPath); err != nil {
		t.Fatalf("sicherheitskopie ungültig: %v", err)
	}
	sc, err := db.Open(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Close()
	var n int
	sc.QueryRow("SELECT COUNT(*) FROM customers WHERE name = 'Nachher'").Scan(&n)
	if n != 1 {
		t.Fatalf("sicherheitskopie enthält den stand vor dem restore nicht")
	}
	// no temporary upload left behind
	left, _ := filepath.Glob(filepath.Join(e.dir, ".restore-*"))
	if len(left) != 0 {
		t.Fatalf("temporäre datei übrig: %v", left)
	}

	// an admin who is not in the backup must sign in again
	other := e.user("other-admin@example.com", "admin")
	st, res = other.restoreUpload(backup)
	if st != 200 || !res.Relogin {
		t.Fatalf("admin ohne konto im backup: %d %+v", st, res)
	}
	expect(t, "alte sitzung ungültig", other.do("GET", "/api/sidebar", nil, nil), 401)
}
