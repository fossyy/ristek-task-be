package handler

import (
	"net/http"
	"ristek-task-be/internal/db/sqlc/repository"
	"ristek-task-be/internal/jwt"
)

type Handler struct {
	repository *repository.Queries
	jwt        *jwt.JWT
}

func New(repository *repository.Queries, jwt *jwt.JWT) *Handler {
	return &Handler{
		repository: repository,
		jwt:        jwt,
	}
}
func badRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(err.Error()))
}

func unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(err.Error()))
}

func (h *Handler) HealthGet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
