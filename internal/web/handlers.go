package web

import (
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/selvakn/radius-server/internal/db"
)

func (s *Server) handleGetLogin(w http.ResponseWriter, r *http.Request) {
	s.renderLogin(w, "")
}

func (s *Server) handlePostLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	admin, ok := s.cfg.FindAdmin(username)
	if !ok || !admin.CheckPassword(password) {
		s.renderLogin(w, "Invalid username or password")
		return
	}

	token := s.sessions.Create(username)
	http.SetCookie(w, &http.Cookie{
		Name:     "rsession",
		Value:    token,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handlePostLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("rsession"); err == nil {
		s.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "rsession", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	sess := sessionFromContext(r.Context())
	s.renderUsers(w, users, sess.CSRFToken, flashFromCookie(r))
	clearFlash(w)
}

func (s *Server) handleGetNewUser(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())
	s.renderForm(w, &db.User{}, false, sess.CSRFToken, "")
}

func (s *Server) handlePostCreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		sess := sessionFromContext(r.Context())
		s.renderForm(w, &db.User{}, false, sess.CSRFToken, "Username and password are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	down, up := parseRates(r)
	if err := s.db.CreateUser(db.User{
		Username:     username,
		PasswordHash: string(hash),
		Enabled:      true,
		DownloadRate: down,
		UploadRate:   up,
	}); err != nil {
		sess := sessionFromContext(r.Context())
		s.renderForm(w, &db.User{}, false, sess.CSRFToken, "Username already exists")
		return
	}

	setFlash(w, "User created", "ok")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleGetEditUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	user, err := s.db.GetUserByID(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	sess := sessionFromContext(r.Context())
	s.renderForm(w, user, true, sess.CSRFToken, "")
}

func (s *Server) handlePostUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByID(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	hash := user.PasswordHash
	if p := r.FormValue("password"); p != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(p), 12)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		hash = string(h)
	}

	down, up := parseRates(r)
	if err := s.db.UpdateUser(id, db.UserUpdate{PasswordHash: hash, DownloadRate: down, UploadRate: up}); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	setFlash(w, "User updated", "ok")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handlePostDisable(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.SetEnabled(id, false); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	setFlash(w, "User disabled", "ok")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handlePostEnable(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.SetEnabled(id, true); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	setFlash(w, "User enabled", "ok")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handlePostDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteUser(id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	setFlash(w, "User deleted", "ok")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func parseID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	return strconv.ParseInt(idStr, 10, 64)
}

func parseRates(r *http.Request) (*int, *int) {
	var down, up *int
	if v := r.FormValue("download_rate"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			down = &n
		}
	}
	if v := r.FormValue("upload_rate"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			up = &n
		}
	}
	return down, up
}
