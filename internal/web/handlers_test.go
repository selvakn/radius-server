package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/selvakn/radius-server/internal/config"
	"github.com/selvakn/radius-server/internal/db"
	"github.com/selvakn/radius-server/internal/web"
)

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
	t.Cleanup(func() { d.Close() })

	cfg := &config.Config{
		Web: config.WebConfig{SessionSecret: "testsecret12345678901234567890"},
		Admins: []config.AdminUser{
			{Username: "admin", PasswordHash: makeHash(t, "adminpass")},
		},
	}
	sessions := web.NewSessionStore()
	srv := web.New(d, cfg, sessions)
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
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
		"download_rate": {"1024"},
		"upload_rate":   {"512"},
		"_csrf":         {sess.CSRFToken},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "rsession", Value: token})
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Errorf("expected redirect after update, got %d: %s", rec.Code, rec.Body.String())
	}
	u2, _ := d.GetUserByUsername("toupdate")
	if u2.DownloadRate == nil || *u2.DownloadRate != 1024 {
		t.Error("expected download rate 1024")
	}
}
