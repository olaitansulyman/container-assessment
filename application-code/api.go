package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type API struct {
	store TodoStore
}

func NewRouter(store TodoStore) http.Handler {
	api := &API{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /ready", api.ready)
	mux.HandleFunc("POST /todos", api.createTodo)
	mux.HandleFunc("GET /todos", api.listTodos)
	mux.HandleFunc("GET /todos/{id}", api.getTodo)
	mux.HandleFunc("PATCH /todos/{id}", api.updateTodo)
	mux.HandleFunc("DELETE /todos/{id}", api.deleteTodo)
	return mux
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "service": "muchtodo-api"})
}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if _, err := a.store.List(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database is not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) createTodo(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string `json:"title"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	todo := &Todo{Title: input.Title}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := a.store.Create(ctx, todo); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create todo")
		return
	}
	writeJSON(w, http.StatusCreated, todo)
}

func (a *API) listTodos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	todos, err := a.store.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list todos")
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

func (a *API) getTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	todo, err := a.store.Get(ctx, id)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not retrieve todo")
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (a *API) updateTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var input struct {
		Title     *string `json:"title"`
		Completed *bool   `json:"completed"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	fields := bson.M{}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		fields["title"] = title
	}
	if input.Completed != nil {
		fields["completed"] = *input.Completed
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "provide title or completed to update")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	todo, err := a.store.Update(ctx, id, fields)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update todo")
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (a *API) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := a.store.Delete(ctx, id); errors.Is(err, mongo.ErrNoDocuments) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete todo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, value string) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid todo id")
		return primitive.NilObjectID, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
