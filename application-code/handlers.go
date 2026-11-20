package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type APIHandler struct {
	repository TodoRepository
}

func NewAPIHandler(repository TodoRepository) *APIHandler {
	return &APIHandler{repository: repository}
}

type todoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func (h *APIHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "service": "muchtodo-api"})
}

func (h *APIHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.repository.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *APIHandler) listTodos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	todos, err := h.repository.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_list_todos")
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

func (h *APIHandler) createTodo(w http.ResponseWriter, r *http.Request) {
	var input todoRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title_is_required")
		return
	}
	now := time.Now().UTC()
	todo := Todo{Title: input.Title, Description: strings.TrimSpace(input.Description), Completed: input.Completed, CreatedAt: now, UpdatedAt: now}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	created, err := h.repository.Create(ctx, todo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_create_todo")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *APIHandler) getTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	todo, err := h.repository.Get(ctx, id)
	if errors.Is(err, ErrTodoNotFound) {
		writeError(w, http.StatusNotFound, "todo_not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_get_todo")
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (h *APIHandler) updateTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var input todoRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title_is_required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	updated, err := h.repository.Update(ctx, id, Todo{
		Title: input.Title, Description: strings.TrimSpace(input.Description),
		Completed: input.Completed, UpdatedAt: time.Now().UTC(),
	})
	if errors.Is(err, ErrTodoNotFound) {
		writeError(w, http.StatusNotFound, "todo_not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_update_todo")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *APIHandler) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.repository.Delete(ctx, id); errors.Is(err, ErrTodoNotFound) {
		writeError(w, http.StatusNotFound, "todo_not_found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_delete_todo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_todo_id")
		return primitive.NilObjectID, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request_must_contain_one_json_object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
