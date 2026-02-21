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
	"github.com/jackc/pgx/v5/pgtype"
)

type QuestionOptionInput struct {
	Label    string `json:"label"`
	Position int32  `json:"position"`
}

type QuestionInput struct {
	Type     string                `json:"type"`
	Title    string                `json:"title"`
	Required bool                  `json:"required"`
	Position int32                 `json:"position"`
	Options  []QuestionOptionInput `json:"options"`
}

type CreateFormRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Questions   []QuestionInput `json:"questions"`
}

type UpdateFormRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Questions   []QuestionInput `json:"questions"`
}

type QuestionOptionResponse struct {
	ID       int32  `json:"id"`
	Label    string `json:"label"`
	Position int32  `json:"position"`
}

type QuestionResponse struct {
	ID       string                   `json:"id"`
	Type     string                   `json:"type"`
	Title    string                   `json:"title"`
	Required bool                     `json:"required"`
	Position int32                    `json:"position"`
	Options  []QuestionOptionResponse `json:"options"`
}

type FormResponse struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	ResponseCount int32              `json:"response_count"`
	Questions     []QuestionResponse `json:"questions"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func optionsByQuestion(options []repository.QuestionOption, questionID uuid.UUID) []QuestionOptionResponse {
	var result []QuestionOptionResponse
	for _, o := range options {
		if o.QuestionID == questionID {
			result = append(result, QuestionOptionResponse{
				ID:       o.ID,
				Label:    o.Label,
				Position: o.Position,
			})
		}
	}
	if result == nil {
		result = []QuestionOptionResponse{}
	}
	return result
}

func buildFormResponse(form repository.Form, questions []repository.Question, options []repository.QuestionOption) FormResponse {
	qs := make([]QuestionResponse, 0, len(questions))
	for _, q := range questions {
		qs = append(qs, QuestionResponse{
			ID:       q.ID.String(),
			Type:     string(q.Type),
			Title:    q.Title,
			Required: q.Required,
			Position: q.Position,
			Options:  optionsByQuestion(options, q.ID),
		})
	}

	desc := ""
	if form.Description.Valid {
		desc = form.Description.String
	}

	return FormResponse{
		ID:            form.ID.String(),
		UserID:        form.UserID.String(),
		Title:         form.Title,
		Description:   desc,
		ResponseCount: form.ResponseCount,
		Questions:     qs,
		CreatedAt:     form.CreatedAt.Time,
		UpdatedAt:     form.UpdatedAt.Time,
	}
}

func (h *Handler) currentUserID(r *http.Request) (uuid.UUID, bool) {
	str, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || str == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

func isChoiceType(t repository.QuestionType) bool {
	switch t {
	case repository.QuestionTypeMultipleChoice,
		repository.QuestionTypeCheckbox,
		repository.QuestionTypeDropdown:
		return true
	}
	return false
}

func (h *Handler) FormsGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(r)
	if !ok {
		unauthorized(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	forms, err := h.repository.ListForms(ctx, userID)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to list forms: %s", err)
		return
	}

	type listItem struct {
		ID            string    `json:"id"`
		Title         string    `json:"title"`
		Description   string    `json:"description"`
		ResponseCount int32     `json:"response_count"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	items := make([]listItem, 0, len(forms))
	for _, f := range forms {
		desc := ""
		if f.Description.Valid {
			desc = f.Description.String
		}
		items = append(items, listItem{
			ID:            f.ID.String(),
			Title:         f.Title,
			Description:   desc,
			ResponseCount: f.ResponseCount,
			CreatedAt:     f.CreatedAt.Time,
			UpdatedAt:     f.UpdatedAt.Time,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(items)
}

func (h *Handler) FormsPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(r)
	if !ok {
		unauthorized(w)
		return
	}

	var req CreateFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Title == "" {
		badRequest(w, errors.New("title is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	form, err := h.repository.CreateForm(ctx, repository.CreateFormParams{
		UserID: userID,
		Title:  req.Title,
		Description: pgtype.Text{
			String: req.Description,
			Valid:  req.Description != "",
		},
	})
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to create form: %s", err)
		return
	}

	questions, options, err := h.saveQuestions(ctx, form.ID, req.Questions)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to save questions: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(buildFormResponse(form, questions, options))
}

func (h *Handler) FormGet(w http.ResponseWriter, r *http.Request) {
	formID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		badRequest(w, errors.New("invalid form id"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	form, err := h.repository.GetFormByID(ctx, formID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		internalServerError(w, err)
		log.Printf("failed to get form: %s", err)
		return
	}

	questions, err := h.repository.GetQuestionsByFormID(ctx, formID)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to get questions: %s", err)
		return
	}

	options, err := h.repository.GetOptionsByFormID(ctx, formID)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to get options: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(buildFormResponse(form, questions, options))
}

func (h *Handler) FormPut(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(r)
	if !ok {
		unauthorized(w)
		return
	}

	formID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		badRequest(w, errors.New("invalid form id"))
		return
	}

	var req UpdateFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Title == "" {
		badRequest(w, errors.New("title is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	form, err := h.repository.GetFormByID(ctx, formID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		internalServerError(w, err)
		return
	}

	if form.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updated, err := h.repository.UpdateForm(ctx, repository.UpdateFormParams{
		ID:    formID,
		Title: req.Title,
		Description: pgtype.Text{
			String: req.Description,
			Valid:  req.Description != "",
		},
	})
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to update form: %s", err)
		return
	}

	if err := h.repository.DeleteOptionsByFormID(ctx, formID); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete options: %s", err)
		return
	}
	if err := h.repository.DeleteQuestionsByFormID(ctx, formID); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete questions: %s", err)
		return
	}

	questions, options, err := h.saveQuestions(ctx, formID, req.Questions)
	if err != nil {
		internalServerError(w, err)
		log.Printf("failed to save questions: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(buildFormResponse(updated, questions, options))
}

func (h *Handler) FormDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(r)
	if !ok {
		unauthorized(w)
		return
	}

	formID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		badRequest(w, errors.New("invalid form id"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	form, err := h.repository.GetFormByID(ctx, formID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		internalServerError(w, err)
		return
	}

	if form.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := h.repository.DeleteForm(ctx, formID); err != nil {
		internalServerError(w, err)
		log.Printf("failed to delete form: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) saveQuestions(ctx context.Context, formID uuid.UUID, inputs []QuestionInput) ([]repository.Question, []repository.QuestionOption, error) {
	var questions []repository.Question
	var options []repository.QuestionOption

	for i, qi := range inputs {
		qt := repository.QuestionType(qi.Type)
		pos := qi.Position
		if pos == 0 {
			pos = int32(i + 1)
		}

		q, err := h.repository.CreateQuestion(ctx, repository.CreateQuestionParams{
			FormID:   formID,
			Type:     qt,
			Title:    qi.Title,
			Required: qi.Required,
			Position: pos,
		})
		if err != nil {
			return nil, nil, err
		}
		questions = append(questions, q)

		if isChoiceType(qt) {
			for j, oi := range qi.Options {
				optPos := oi.Position
				if optPos == 0 {
					optPos = int32(j + 1)
				}
				opt, err := h.repository.CreateQuestionOption(ctx, repository.CreateQuestionOptionParams{
					FormID:     formID,
					QuestionID: q.ID,
					Label:      oi.Label,
					Position:   optPos,
				})
				if err != nil {
					return nil, nil, err
				}
				options = append(options, opt)
			}
		}
	}

	return questions, options, nil
}
