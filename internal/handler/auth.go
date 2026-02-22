package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"ristek-task-be/internal/db/sqlc/repository"
	"ristek-task-be/internal/middleware"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func isDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// RegisterPost registers a new user account
//
//	@Summary		Register a new user
//	@Description	Create a new user account with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		Auth	true	"Register credentials"
//	@Success		201
//	@Failure		400	{string}	string	"Bad request (e.g. email already exists, password too short)"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/api/auth/register [post]
func (h *Handler) RegisterPost(w http.ResponseWriter, r *http.Request) {
	var register Auth
	if err := json.NewDecoder(r.Body).Decode(&register); err != nil {
		badRequest(w, err)
		log.Printf("failed to decode request body: %s", err)
		return
	}

	if register.Email == "" || register.Password == "" {
		badRequest(w, errors.New("email and password are required"))
		return
	}

	if len(register.Password) < 8 {
		badRequest(w, errors.New("password must be at least 8 characters long"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(register.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to hash password: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.repository.CreateUser(ctx, repository.CreateUserParams{
		Email:        register.Email,
		PasswordHash: string(hashedPassword),
	})

	if err != nil {
		if isDuplicateError(err) {
			badRequest(w, errors.New("email already exists"))
			return
		}
		internalServerError(w, err)
		log.Printf("failed to create user: %s", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshPost refreshes the access token using a refresh token
//
//	@Summary		Refresh access token
//	@Description	Exchange a valid refresh token for a new access token and refresh token pair
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshRequest	true	"Refresh token payload"
//	@Success		200		{object}	map[string]interface{}	"access_token, refresh_token, expires_in"
//	@Failure		400		{string}	string					"Bad request"
//	@Failure		401		{string}	string					"Unauthorized (invalid or expired refresh token)"
//	@Failure		500		{string}	string					"Internal server error"
//	@Router			/api/auth/refresh [post]
func (h *Handler) RefreshPost(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		log.Printf("failed to decode request body: %s", err)
		return
	}

	if req.RefreshToken == "" {
		badRequest(w, errors.New("refresh_token is required"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rt, err := h.repository.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			unauthorized(w)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get refresh token: %s", err)
		return
	}

	if rt.ExpiresAt.Time.Before(time.Now()) {
		unauthorized(w)
		return
	}

	user, err := h.repository.GetUserByID(ctx, rt.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			unauthorized(w)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get user: %s", err)
		return
	}

	if err := h.repository.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete old refresh token: %s", err)
		return
	}

	accessToken, err := h.jwt.GenerateAccessToken(user.ID.String(), user.Email)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to generate access token: %s", err)
		return
	}

	rawRefreshToken := uuid.New().String()
	_, err = h.repository.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     rawRefreshToken,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to create refresh token: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"refresh_token": rawRefreshToken,
		"expires_in":    900,
	})
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutPost logs out the current session by invalidating the refresh token
//
//	@Summary		Logout
//	@Description	Invalidate the given refresh token to log out the current session
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	LogoutRequest	true	"Refresh token to invalidate"
//	@Success		204
//	@Failure		400	{string}	string	"Bad request"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/api/auth/logout [post]
func (h *Handler) LogoutPost(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		log.Printf("failed to decode request body: %s", err)
		return
	}

	if req.RefreshToken == "" {
		badRequest(w, errors.New("refresh_token is required"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.repository.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete refresh token: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LogoutAllDelete logs out all sessions for the authenticated user
//
//	@Summary		Logout all sessions
//	@Description	Invalidate all refresh tokens for the authenticated user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204
//	@Failure		401	{string}	string	"Unauthorized"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/api/auth/logout/all [delete]
func (h *Handler) LogoutAllDelete(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		unauthorized(w)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		unauthorized(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.repository.DeleteUserRefreshTokens(ctx, userID); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete user refresh tokens: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MeGet returns the profile of the currently authenticated user
//
//	@Summary		Get current user
//	@Description	Return profile information for the authenticated user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"id, email, created_at, updated_at"
//	@Failure		401	{string}	string					"Unauthorized"
//	@Failure		500	{string}	string					"Internal server error"
//	@Router			/api/auth/me [get]
func (h *Handler) MeGet(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		unauthorized(w)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		unauthorized(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.repository.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			unauthorized(w)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get user: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":         user.ID.String(),
		"email":      user.Email,
		"created_at": user.CreatedAt.Time,
		"updated_at": user.UpdatedAt.Time,
	})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// MePasswordPatch changes the password of the authenticated user
//
//	@Summary		Change password
//	@Description	Update the password of the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	ChangePasswordRequest	true	"Old and new password"
//	@Success		204
//	@Failure		400	{string}	string	"Bad request (e.g. incorrect old password)"
//	@Failure		401	{string}	string	"Unauthorized"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/api/auth/me/password [patch]
func (h *Handler) MePasswordPatch(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		unauthorized(w)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		unauthorized(w)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		log.Printf("failed to decode request body: %s", err)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		badRequest(w, errors.New("old_password and new_password are required"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.repository.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			unauthorized(w)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get user: %s", err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		badRequest(w, errors.New("incorrect old password"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to hash password: %s", err)
		return
	}

	_, err = h.repository.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to update password: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LoginPost authenticates a user and returns tokens
//
//	@Summary		Login
//	@Description	Authenticate with email and password to receive access and refresh tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		Auth	true	"Login credentials"
//	@Success		200		{object}	map[string]interface{}	"access_token, refresh_token, expires_in"
//	@Failure		400		{string}	string					"Bad request"
//	@Failure		401		{string}	string					"Unauthorized (invalid credentials)"
//	@Failure		500		{string}	string					"Internal server error"
//	@Router			/api/auth/login [post]
func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
	var login Auth
	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		badRequest(w, err)
		log.Printf("failed to decode request body: %s", err)
		return
	}

	if login.Email == "" || login.Password == "" {
		badRequest(w, errors.New("email and password are required"))
		return
	}

	user, err := h.repository.GetUserByEmail(r.Context(), login.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			unauthorized(w)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get user by email: %s", err)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(login.Password),
	); err != nil {
		unauthorized(w)
		return
	}

	accessToken, err := h.jwt.GenerateAccessToken(user.ID.String(), user.Email)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to generate access token: %s", err)
		return
	}

	rawRefreshToken := uuid.New().String()
	_, err = h.repository.CreateRefreshToken(r.Context(), repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     rawRefreshToken,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to create refresh token: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"refresh_token": rawRefreshToken,
		"expires_in":    900,
	})
}
