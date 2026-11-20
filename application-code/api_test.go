package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type memoryStore struct {
	items map[primitive.ObjectID]Todo
}

func newMemoryStore() *memoryStore { return &memoryStore{items: make(map[primitive.ObjectID]Todo)} }
func (s *memoryStore) Create(_ context.Context, todo *Todo) error {
	todo.ID = primitive.NewObjectID()
	s.items[todo.ID] = *todo
	return nil
}
func (s *memoryStore) List(_ context.Context) ([]Todo, error) {
	items := make([]Todo, 0, len(s.items))
	for _, item := range s.items { items = append(items, item) }
	return items, nil
}
func (s *memoryStore) Get(_ context.Context, id primitive.ObjectID) (*Todo, error) {
	item, ok := s.items[id]; if !ok { return nil, mongo.ErrNoDocuments }; return &item, nil
}
func (s *memoryStore) Update(_ context.Context, id primitive.ObjectID, fields bson.M) (*Todo, error) {
	item, ok := s.items[id]; if !ok { return nil, mongo.ErrNoDocuments }
	if title, ok := fields["title"].(string); ok { item.Title = title }
	if completed, ok := fields["completed"].(bool); ok { item.Completed = completed }
	s.items[id] = item
	return &item, nil
}
func (s *memoryStore) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := s.items[id]; !ok { return mongo.ErrNoDocuments }; delete(s.items, id); return nil
}

func TestCreateTodo(t *testing.T) {
	store := newMemoryStore()
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"  Write tests  "}`))
	rec := httptest.NewRecorder()
	NewRouter(store).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated { t.Fatalf("got %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String()) }
	var got Todo
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if got.Title != "Write tests" { t.Fatalf("title = %q", got.Title) }
}

func TestCreateTodoRejectsEmptyTitle(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"  "}`))
	rec := httptest.NewRecorder()
	NewRouter(newMemoryStore()).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("got %d, want 400", rec.Code) }
}

func TestTodoCRUD(t *testing.T) {
	store := newMemoryStore()
	handler := NewRouter(store)
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"Initial"}`)))
	var todo Todo
	if err := json.Unmarshal(create.Body.Bytes(), &todo); err != nil { t.Fatal(err) }

	update := httptest.NewRecorder()
	handler.ServeHTTP(update, httptest.NewRequest(http.MethodPatch, "/todos/"+todo.ID.Hex(), strings.NewReader(`{"completed":true}`)))
	if update.Code != http.StatusOK { t.Fatalf("PATCH got %d: %s", update.Code, update.Body.String()) }

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/todos/"+todo.ID.Hex(), nil))
	if get.Code != http.StatusOK { t.Fatalf("GET got %d", get.Code) }

	del := httptest.NewRecorder()
	handler.ServeHTTP(del, httptest.NewRequest(http.MethodDelete, "/todos/"+todo.ID.Hex(), nil))
	if del.Code != http.StatusNoContent { t.Fatalf("DELETE got %d", del.Code) }

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/todos/"+todo.ID.Hex(), nil))
	if missing.Code != http.StatusNotFound { t.Fatalf("missing GET got %d", missing.Code) }
}

func TestInvalidID(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(newMemoryStore()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/todos/not-an-id", nil))
	if rec.Code != http.StatusBadRequest { t.Fatalf("got %d, want 400", rec.Code) }
}
