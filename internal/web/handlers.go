package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/selvakn/radius-server/internal/auth"
	"github.com/selvakn/radius-server/internal/db"
)

func (s *Server) handleGetAttempts(w http.ResponseWriter, r *http.Request) {
	attempts, err := s.db.ListAttemptSummaries()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	sess := sessionFromContext(r.Context())
	s.renderLayout(w, "attempts.html", pageData{Attempts: attempts, CSRFToken: sess.CSRFToken})
}

func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	var sessions []db.Session
	var err error
	if username != "" {
		sessions, err = s.db.ListSessionsByUser(username)
	} else {
		sessions, err = s.db.ListRecentSessions(200)
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	sess := sessionFromContext(r.Context())
	data := pageData{Sessions: sessions, Username: username, CSRFToken: sess.CSRFToken}
	s.renderLayout(w, "sessions.html", data)
}

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
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure omitted: admin UI runs on HTTP within trusted networks
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
	http.SetCookie(w, &http.Cookie{Name: "rsession", MaxAge: -1, Path: "/"}) //nolint:gosec
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	online, err := s.db.OnlineUsernames()
	if err != nil {
		online = map[string]bool{}
	}
	sess := sessionFromContext(r.Context())
	s.renderUsers(w, users, online, sess.CSRFToken, flashFromCookie(r))
	clearFlash(w)
}

func (s *Server) handleGetNewUser(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())
	u := &db.User{Username: r.URL.Query().Get("username")}
	s.renderForm(w, u, false, sess.CSRFToken, "")
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

	bcryptHash, ntHash, err := hashPassword(password)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	down, up := parseRates(r)
	if err := s.db.CreateUser(db.User{
		Username:     username,
		PasswordHash: bcryptHash,
		NTHash:       ntHash,
		Enabled:      true,
		DownloadRate: down,
		UploadRate:   up,
	}); err != nil {
		if existing, lookupErr := s.db.GetUserByUsername(username); lookupErr == nil {
			setFlash(w, "User already exists", "err")
			http.Redirect(w, r, fmt.Sprintf("/users/%d/edit", existing.ID), http.StatusSeeOther) //nolint:gosec // redirect target is an int64 DB ID, not user-controlled
			return
		}
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

	bcryptHash := user.PasswordHash
	ntHash := user.NTHash
	if p := r.FormValue("password"); p != "" {
		var err error
		bcryptHash, ntHash, err = hashPassword(p)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	down, up := parseRates(r)
	if err := s.db.UpdateUser(id, db.UserUpdate{PasswordHash: bcryptHash, NTHash: ntHash, DownloadRate: down, UploadRate: up}); err != nil {
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

func hashPassword(plain string) (bcryptHash, ntHash string, err error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
	if err != nil {
		return "", "", err
	}
	nt, err := auth.NTHash(plain)
	if err != nil {
		return "", "", err
	}
	return string(h), nt, nil
}

func (s *Server) handlePostDisconnectSession(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	session, err := s.db.GetActiveSessionByID(id)
	if err != nil {
		setFlash(w, "Session not found or already stopped", "err")
		http.Redirect(w, r, "/sessions", http.StatusSeeOther)
		return
	}
	if session.NasIP == "" {
		setFlash(w, "Cannot disconnect: no NAS IP for this session", "err")
		http.Redirect(w, r, "/sessions", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.coa.SendDisconnect(ctx, session.NasIP, s.cfg.Radius.SharedSecret, session.SessionID, session.Username); err != nil {
		setFlash(w, fmt.Sprintf("Disconnect failed: %v", err), "err")
		http.Redirect(w, r, "/sessions", http.StatusSeeOther)
		return
	}

	_ = s.db.StopSession(session.SessionID, session.BytesIn, session.BytesOut, session.SessionTime, "Admin-Request", time.Now())
	setFlash(w, fmt.Sprintf("Session for %s disconnected", session.Username), "ok")
	http.Redirect(w, r, "/sessions", http.StatusSeeOther)
}

func (s *Server) handlePostDisconnectAllSessions(w http.ResponseWriter, r *http.Request) {
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
	activeSessions, err := s.db.GetActiveSessionsByUser(user.Username)
	if err != nil || len(activeSessions) == 0 {
		setFlash(w, "No active sessions to disconnect", "ok")
		http.Redirect(w, r, fmt.Sprintf("/users/%d/edit", id), http.StatusSeeOther) //nolint:gosec
		return
	}

	ok, failed := 0, 0
	for _, sess := range activeSessions {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		disconnectErr := s.coa.SendDisconnect(ctx, sess.NasIP, s.cfg.Radius.SharedSecret, sess.SessionID, sess.Username)
		cancel()
		if disconnectErr == nil {
			_ = s.db.StopSession(sess.SessionID, sess.BytesIn, sess.BytesOut, sess.SessionTime, "Admin-Request", time.Now())
			ok++
		} else {
			failed++
		}
	}

	msg := fmt.Sprintf("Disconnected %d of %d sessions", ok, ok+failed)
	setFlash(w, msg, "ok")
	http.Redirect(w, r, fmt.Sprintf("/users/%d/edit", id), http.StatusSeeOther) //nolint:gosec
}

func parseID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	return strconv.ParseInt(idStr, 10, 64)
}

func parseRates(r *http.Request) (*int, *int) {
	var down, up *int
	if v := r.FormValue("download_rate"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 1 && n <= 500 {
			kbps := n * 1000
			down = &kbps
		}
	}
	if v := r.FormValue("upload_rate"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 1 && n <= 500 {
			kbps := n * 1000
			up = &kbps
		}
	}
	return down, up
}
