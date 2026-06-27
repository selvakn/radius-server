package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/radius-server/internal/config"
	"github.com/selvakn/radius-server/internal/db"
)

//go:embed templates/*.html
var templateFS embed.FS

type Server struct {
	db       *db.DB
	cfg      *config.Config
	sessions *SessionStore
	router   *chi.Mux
}

var tmplFuncs = template.FuncMap{
	"deref": func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	},
	"not": func(b bool) bool { return !b },
	"mbps": func(p *int) int {
		if p == nil {
			return 0
		}
		return *p / 1000
	},
	"min": func(a, b int) int {
		if a < b {
			return a
		}
		return b
	},
	"slice": func(s string, i, j int) string { return s[i:j] },
	"fmtbytes": func(b int64) string {
		if b == 0 {
			return "—"
		}
		const unit = 1024
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB", math.Round(float64(b)/float64(div)*10)/10, "KMGT"[exp])
	},
	"fmtduration": func(secs int64) string {
		if secs == 0 {
			return "—"
		}
		h := secs / 3600
		m := (secs % 3600) / 60
		s := secs % 60
		if h > 0 {
			return fmt.Sprintf("%dh%02dm", h, m)
		}
		if m > 0 {
			return fmt.Sprintf("%dm%02ds", m, s)
		}
		return fmt.Sprintf("%ds", s)
	},
	"fmttime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("01-02 15:04")
	},
}

func New(database *db.DB, cfg *config.Config, sessions *SessionStore) *Server {
	s := &Server{
		db:       database,
		cfg:      cfg,
		sessions: sessions,
	}
	s.router = s.buildRouter()
	return s
}

func (s *Server) parseTemplate(files ...string) (*template.Template, error) {
	return template.New("").Funcs(tmplFuncs).ParseFS(templateFS, files...)
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/login", s.handleGetLogin)
	r.Post("/login", s.handlePostLogin)

	r.Group(func(r chi.Router) {
		r.Use(s.sessionMiddleware)
		r.Post("/logout", s.handlePostLogout)
		r.Get("/", s.handleGetUsers)
		r.Get("/sessions", s.handleGetSessions)
		r.Get("/users/new", s.handleGetNewUser)
		r.With(s.csrfMiddleware).Post("/users", s.handlePostCreateUser)
		r.Get("/users/{id}/edit", s.handleGetEditUser)
		r.With(s.csrfMiddleware).Post("/users/{id}", s.handlePostUpdateUser)
		r.With(s.csrfMiddleware).Post("/users/{id}/disable", s.handlePostDisable)
		r.With(s.csrfMiddleware).Post("/users/{id}/enable", s.handlePostEnable)
		r.With(s.csrfMiddleware).Post("/users/{id}/delete", s.handlePostDelete)
	})

	return r
}

type contextKey string

const sessionKey contextKey = "session"

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("rsession")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		sess, ok := s.sessions.Get(c.Value)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), sessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := sessionFromContext(r.Context())
		if sess == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		token := r.FormValue("_csrf")
		if token == "" || token != sess.CSRFToken {
			http.Error(w, "forbidden: invalid CSRF token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sessionFromContext(ctx context.Context) *Session {
	if sess, ok := ctx.Value(sessionKey).(*Session); ok {
		return sess
	}
	return nil
}

type pageData struct {
	Users     []db.User
	User      *db.User
	Edit      bool
	CSRFToken string
	Flash     string
	FlashType string
	Sessions  []db.Session
	Username  string
}

func (s *Server) renderLogin(w http.ResponseWriter, errMsg string) {
	t, err := s.parseTemplate("templates/login.html")
	if err != nil {
		http.Error(w, "template error", 500)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = t.ExecuteTemplate(w, "login.html", map[string]string{"Error": errMsg})
}

func (s *Server) renderUsers(w http.ResponseWriter, users []db.User, csrf, flash string) {
	flashType := "ok"
	if flash == "" {
		flashType = ""
	}
	data := pageData{Users: users, CSRFToken: csrf, Flash: flash, FlashType: flashType}
	s.renderLayout(w, "users.html", data)
}

func (s *Server) renderForm(w http.ResponseWriter, user *db.User, edit bool, csrf, errMsg string) {
	data := pageData{User: user, Edit: edit, CSRFToken: csrf, Flash: errMsg, FlashType: "err"}
	if errMsg == "" {
		data.FlashType = ""
	}
	s.renderLayout(w, "user_form.html", data)
}

func (s *Server) renderLayout(w http.ResponseWriter, contentFile string, data pageData) {
	t, err := s.parseTemplate("templates/layout.html", "templates/"+contentFile)
	if err != nil {
		http.Error(w, "template error", 500)
		return
	}
	_ = t.ExecuteTemplate(w, "layout.html", data)
}

func setFlash(w http.ResponseWriter, msg, typ string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // flash cookie carries no sensitive data
		Name:    "flash",
		Value:   typ + ":" + msg,
		Path:    "/",
		Expires: time.Now().Add(5 * time.Second),
	})
}

func flashFromCookie(r *http.Request) string {
	c, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	if len(c.Value) > 3 && c.Value[2] == ':' {
		return c.Value[3:]
	}
	return c.Value
}

func clearFlash(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "flash", MaxAge: -1, Path: "/"}) //nolint:gosec
}
