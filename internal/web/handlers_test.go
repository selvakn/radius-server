package web_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/selvakn/radius-server/internal/config"
	"github.com/selvakn/radius-server/internal/db"
	"github.com/selvakn/radius-server/internal/web"
)

type mockCoA struct{ err error }

func (m *mockCoA) SendDisconnect(_ context.Context, _, _, _, _ string) error { return m.err }

// sessionCookie builds a bare session cookie for test requests.
// Real browsers receive this via Set-Cookie; Secure is irrelevant in unit tests.
func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "rsession", Value: token} //nolint:gosec // test-only cookie, no browser security attributes needed
}

func makeHash(t *testing.T, pass string) string {
	t.Helper()
	h, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	return string(h)
}

func setupServer(t *testing.T) (*web.Server, *db.DB, *web.SessionStore) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	cfg := &config.Config{
		Web: config.WebConfig{SessionSecret: "testsecret12345678901234567890"},
		Admins: []config.AdminUser{
			{Username: "admin", PasswordHash: makeHash(t, "adminpass")},
		},
	}
	sessions := web.NewSessionStore()
	srv := web.New(d, cfg, sessions, &mockCoA{})
	return srv, d, sessions
}

func TestLogin_ValidCredentials(t *testing.T) {
	srv, _, _ := setupServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{
		"username": {"admin"},
		"password": {"adminpass"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "rsession" {
			found = true
		}
	}
	if !found {
		t.Error("expected rsession cookie to be set")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	srv, _, _ := setupServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{
		"username": {"admin"},
		"password": {"wrongpass"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Router().ServeHTTP(rec, req)
	if rec.Code == http.StatusSeeOther || rec.Code == http.StatusFound {
		t.Error("expected non-redirect for invalid credentials")
	}
}

func TestUsersIndex_Unauthenticated(t *testing.T) {
	srv, _, _ := setupServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect to login, got %d", rec.Code)
	}
}

func TestUsersIndex_Authenticated(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCreateUser_Success(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users", strings.NewReader(url.Values{
		"username": {"newuser"},
		"password": {"newpass"},
		"_csrf":    {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect after create, got %d: %s", rec.Code, rec.Body.String())
	}
	_, err := d.GetUserByUsername("newuser")
	if err != nil {
		t.Errorf("user should exist in DB: %v", err)
	}
}

func TestDisableUser(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)

	_ = d.CreateUser(db.User{Username: "victim", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("victim")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d/disable", u.ID), strings.NewReader(url.Values{
		"_csrf": {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	u2, _ := d.GetUserByUsername("victim")
	if u2.Enabled {
		t.Error("expected user to be disabled")
	}
}

func TestDeleteUser(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)

	_ = d.CreateUser(db.User{Username: "todelete", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("todelete")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d/delete", u.ID), strings.NewReader(url.Values{
		"_csrf": {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	_, err := d.GetUserByUsername("todelete")
	if err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestCSRF_MissingToken(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users", strings.NewReader(url.Values{
		"username": {"newuser"},
		"password": {"pass"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing CSRF, got %d", rec.Code)
	}
}

func TestGetLogin_Renders(t *testing.T) {
	srv, _, _ := setupServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "RADIUS Admin") {
		t.Error("expected login page content")
	}
}

func TestLogout(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect after logout, got %d", rec.Code)
	}
	_, ok := sessions.Get(token)
	if ok {
		t.Error("expected session to be deleted after logout")
	}
}

func TestGetNewUser_Renders(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/new", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetEditUser_Renders(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")

	_ = d.CreateUser(db.User{Username: "editable", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("editable")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/users/%d/edit", u.ID), nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestEnableUser(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)

	_ = d.CreateUser(db.User{Username: "tooenable", PasswordHash: "h", Enabled: false})
	u, _ := d.GetUserByUsername("tooenable")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d/enable", u.ID), strings.NewReader(url.Values{
		"_csrf": {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	u2, _ := d.GetUserByUsername("tooenable")
	if !u2.Enabled {
		t.Error("expected user to be enabled")
	}
}

func TestUpdateUser(t *testing.T) {
	srv, d, sessions := setupServer(t)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)

	_ = d.CreateUser(db.User{Username: "toupdate", PasswordHash: makeHash(t, "oldpass"), Enabled: true})
	u, _ := d.GetUserByUsername("toupdate")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d", u.ID), strings.NewReader(url.Values{
		"password":      {"newpass"},
		"download_rate": {"2"},
		"upload_rate":   {"1"},
		"_csrf":         {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect after update, got %d: %s", rec.Code, rec.Body.String())
	}
	u2, _ := d.GetUserByUsername("toupdate")
	if u2.DownloadRate == nil || *u2.DownloadRate != 2000 {
		t.Errorf("expected download rate 2000 kbps, got %v", u2.DownloadRate)
	}
}

func TestGetAttempts_Unauthenticated(t *testing.T) {
	srv, _, _ := setupServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/attempts", nil)
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect to login, got %d", rec.Code)
	}
}

func TestGetAttempts_Authenticated(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/attempts", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetAttempts_EmptyState(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/attempts", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "no authentication attempts") {
		t.Error("expected empty state message")
	}
}

func TestGetAttempts_AddButtonOnlyForUnknown(t *testing.T) {
	srv, d, sessions := setupServer(t)
	_ = d.CreateUser(db.User{Username: "knownuser", PasswordHash: "h", Enabled: true})
	_ = d.RecordAttempt("knownuser", "accepted", "")
	_ = d.RecordAttempt("unknownuser", "rejected", "")
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/attempts", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "add-from-attempt") && strings.Count(body, "add-from-attempt") != 1 {
		t.Error("expected exactly 1 add button (for unknown user only)")
	}
}

func TestGetNewUser_UsernameQueryParam(t *testing.T) {
	srv, _, sessions := setupServer(t)
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/new?username=prefilled", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "prefilled") {
		t.Error("expected pre-filled username in form")
	}
}

func TestCreateUser_DuplicateRedirectsToEdit(t *testing.T) {
	srv, d, sessions := setupServer(t)
	_ = d.CreateUser(db.User{Username: "existing", PasswordHash: makeHash(t, "pass"), Enabled: true})
	existing, _ := d.GetUserByUsername("existing")
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users", strings.NewReader(url.Values{
		"username": {"existing"},
		"password": {"anypass"},
		"_csrf":    {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	location := rec.Header().Get("Location")
	if !strings.Contains(location, fmt.Sprintf("/users/%d/edit", existing.ID)) {
		t.Errorf("expected redirect to edit page, got location: %q", location)
	}
}

func setupServerWithCoA(t *testing.T, coaErr error) (*web.Server, *db.DB, *web.SessionStore) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	cfg := &config.Config{
		Radius: config.RadiusConfig{SharedSecret: "secret"},
		Web:    config.WebConfig{SessionSecret: "testsecret12345678901234567890"},
		Admins: []config.AdminUser{{Username: "admin", PasswordHash: makeHash(t, "adminpass")}},
	}
	sessions := web.NewSessionStore()
	srv := web.New(d, cfg, sessions, &mockCoA{err: coaErr})
	return srv, d, sessions
}

func TestDisconnectSession_NotFound(t *testing.T) {
	srv, _, sessions := setupServerWithCoA(t, nil)
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sessions/9999/disconnect", strings.NewReader(url.Values{
		"_csrf": {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
}

func TestDisconnectSession_NoNasIP(t *testing.T) {
	srv, d, sessions := setupServerWithCoA(t, nil)
	_ = d.UpsertSessionStart("sess-nonas", "alice", "", "", time.Now())
	active, _ := d.ListActiveSessions()
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/sessions/%d/disconnect", active[0].ID),
		strings.NewReader(url.Values{"_csrf": {sess.CSRFToken}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
}

func TestDisconnectSession_Success(t *testing.T) {
	srv, d, sessions := setupServerWithCoA(t, nil)
	_ = d.UpsertSessionStart("sess-ok", "bob", "10.0.0.1", "", time.Now())
	active, _ := d.ListActiveSessions()
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/sessions/%d/disconnect", active[0].ID),
		strings.NewReader(url.Values{"_csrf": {sess.CSRFToken}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect after success, got %d", rec.Code)
	}
	remaining, _ := d.ListActiveSessions()
	if len(remaining) != 0 {
		t.Error("expected session to be marked stopped after successful disconnect")
	}
}

func TestDisconnectAll_NoActiveSessions(t *testing.T) {
	srv, d, sessions := setupServerWithCoA(t, nil)
	_ = d.CreateUser(db.User{Username: "carol", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("carol")
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d/disconnect-all", u.ID),
		strings.NewReader(url.Values{"_csrf": {sess.CSRFToken}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
}

func TestDisconnectAll_Success(t *testing.T) {
	srv, d, sessions := setupServerWithCoA(t, nil)
	_ = d.CreateUser(db.User{Username: "dave", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("dave")
	_ = d.UpsertSessionStart("da1", "dave", "10.0.0.1", "", time.Now())
	_ = d.UpsertSessionStart("da2", "dave", "10.0.0.1", "", time.Now())
	token := sessions.Create("admin")
	sess, _ := sessions.Get(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/users/%d/disconnect-all", u.ID),
		strings.NewReader(url.Values{"_csrf": {sess.CSRFToken}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
	remaining, _ := d.GetActiveSessionsByUser("dave")
	if len(remaining) != 0 {
		t.Errorf("expected all sessions stopped, got %d active", len(remaining))
	}
}

func TestUsersPage_ShowsCurrentMonthUsage(t *testing.T) {
	srv, d, sessions := setupServer(t)
	_ = d.CreateUser(db.User{Username: "usageuser", PasswordHash: "h", Enabled: true})
	_ = d.UpsertSessionStart("usg1", "usageuser", "10.0.0.1", "", time.Now())
	_ = d.UpdateSessionInterim("usg1", 1024000, 2048000, 60)
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "usageuser") {
		t.Error("expected usageuser in users page")
	}
}

func TestEditUser_ShowsMonthlyHistory(t *testing.T) {
	srv, d, sessions := setupServer(t)
	_ = d.CreateUser(db.User{Username: "histuser", PasswordHash: "h", Enabled: true})
	u, _ := d.GetUserByUsername("histuser")
	_ = d.UpsertSessionStart("hst1", "histuser", "10.0.0.1", "", time.Now())
	_ = d.StopSession("hst1", 500, 1000, 30, "User-Request", time.Now())
	token := sessions.Create("admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/users/%d/edit", u.ID), nil)
	req.AddCookie(sessionCookie(token))
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
