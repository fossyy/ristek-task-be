package server

import (
	"fmt"
	"net/http"
	_ "ristek-task-be/docs"
	"ristek-task-be/internal/db/sqlc/repository"
	"ristek-task-be/internal/handler"
	"ristek-task-be/internal/jwt"
	"ristek-task-be/internal/middleware"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	addr       string
	port       string
	repository *repository.Queries
	jwt        *jwt.JWT
}

func New(addr string, port string, repository *repository.Queries, jwt *jwt.JWT) *Server {
	return &Server{
		addr:       addr,
		port:       port,
		repository: repository,
		jwt:        jwt,
	}
}

func router(repository *repository.Queries, jwt *jwt.JWT) *http.ServeMux {
	r := http.NewServeMux()
	h := handler.New(repository, jwt)

	r.HandleFunc("GET /health", h.HealthGet)
	r.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	authRoute := http.NewServeMux()
	r.Handle("/api/auth/", http.StripPrefix("/api/auth", authRoute))
	authRoute.HandleFunc("POST /register", h.RegisterPost)
	authRoute.HandleFunc("POST /login", h.LoginPost)
	authRoute.HandleFunc("POST /refresh", h.RefreshPost)
	authRoute.HandleFunc("POST /logout", h.LogoutPost)
	authRoute.Handle("DELETE /logout/all", middleware.Auth(jwt)(http.HandlerFunc(h.LogoutAllDelete)))
	authRoute.Handle("GET /me", middleware.Auth(jwt)(http.HandlerFunc(h.MeGet)))
	authRoute.Handle("PATCH /me/password", middleware.Auth(jwt)(http.HandlerFunc(h.MePasswordPatch)))

	formRoute := http.NewServeMux()
	r.Handle("/api/forms", middleware.Auth(jwt)(http.HandlerFunc(h.FormsGet)))
	r.Handle("/api/form/", http.StripPrefix("/api/form", formRoute))
	formRoute.HandleFunc("GET /{id}", h.FormGet)
	formRoute.Handle("POST /{$}", middleware.Auth(jwt)(http.HandlerFunc(h.FormsPost)))
	formRoute.Handle("PUT /{id}", middleware.Auth(jwt)(http.HandlerFunc(h.FormPut)))
	formRoute.Handle("DELETE /{id}", middleware.Auth(jwt)(http.HandlerFunc(h.FormDelete)))

	formRoute.HandleFunc("POST /{id}/response", h.FormResponsesPost)
	formRoute.Handle("GET /{id}/responses", middleware.Auth(jwt)(http.HandlerFunc(h.FormResponsesGet)))

	return r
}

func (s *Server) Start() error {
	r := router(s.repository, s.jwt)
	hs := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", s.addr, s.port),
		Handler: middleware.Handler(middleware.CORS(r)),
	}

	return hs.ListenAndServe()
}
