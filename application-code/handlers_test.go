package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fakeTodoRepository struct {
	todos map[primitive.ObjectID]Todo
	pingErr error
}

func newFakeRepository() *fakeTodoRepository {
	return &fakeTodoRepository{todos: make(map[primitive.ObjectID]Todo)}
}
func (f *fakeTodoRepository) List(context.Context) ([]Todo, error) {
	items := make([]Todo, 0, len(f.todos))
	for _, item := range f.todos { items = append(items, item) }
	return items, nil
}
func (f *fakeTodoRepository) Create(_ context.Context, todo Todo) (Todo, error) {
	todo.ID = primitive.NewObjectID()
	f.todos[todo.ID] = todo
	return todo, nil
}
func (f *fakeTodoRepository) Get(_ context.Context, id primitive.ObjectID) (Todo, error) {
	item, ok := f.todos[id]
	if !ok { return Todo{}, ErrTodoNotFound }
	return item, nil
}
func (f *fakeTodoRepository) Update(_ context.Context, id primitive.ObjectID, todo Todo) (Todo, error) {
	if _, ok := f.todos[id]; !ok { return Todo{}, ErrTodoNotFound }
	todo.ID = id
	f.todos[id] = todo
	return todo, nil
}
func (f *fakeTodoRepository) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := f.todos[id]; !ok { return ErrTodoNotFound }
	delete(f.todos, id)
	return nil
}
func (f *fakeTodoRepository) Ping(context.Context) error { return f.pingErr }

func TestCreateTodoRequiresTitle(t *testing.T) {
	h := NewAPIHandler(newFakeRepository())
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"  "}`))
	res := httptest.NewRecorder()
	h.createTodo(res, req)
	if res.Code != http.StatusBadRequest { t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest) }
}

func TestCreateAndGetTodo(t *testing.T) {
	repo := newFakeRepository()
	h := NewAPIHandler(repo)
	createReq := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"  Learn Go  ","description":"API practice"}`))
	createRes := httptest.NewRecorder()
	h.createTodo(createRes, createReq)
	if createRes.Code != http.StatusCreated { t.Fatalf("create status = %d, body=%s", createRes.Code, createRes.Body.String()) }
	if len(repo.todos) != 1 { t.Fatalf("stored todos = %d, want 1", len(repo.todos)) }
	for _, item := range repo.todos {
		if item.Title != "Learn Go" { t.Fatalf("title = %q, want trimmed title", item.Title) }
		if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() { t.Fatal("timestamps must be set") }
	}
}

func TestReadyReturnsUnavailableWhenDatabasePingFails(t *testing.T) {
	repo := newFakeRepository()
	repo.pingErr = errors.New("database offline")
	h := NewAPIHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()
	h.ready(res, req)
	if res.Code != http.StatusServiceUnavailable { t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable) }
}

func TestInvalidTodoID(t *testing.T) {
	h := NewAPIHandler(newFakeRepository())
	req := httptest.NewRequest(http.MethodGet, "/todos/not-an-object-id", nil)
	req.SetPathValue("id", "not-an-object-id")
	res := httptest.NewRecorder()
	h.getTodo(res, req)
	if res.Code != http.StatusBadRequest { t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest) }
}

func TestDeleteMissingTodo(t *testing.T) {
	h := NewAPIHandler(newFakeRepository())
	id := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodDelete, "/todos/"+id.Hex(), nil)
	req.SetPathValue("id", id.Hex())
	res := httptest.NewRecorder()
	h.deleteTodo(res, req)
	if res.Code != http.StatusNotFound { t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound) }
}

func TestHealthDoesNotDependOnDatabase(t *testing.T) {
	repo := newFakeRepository()
	repo.pingErr = errors.New("database offline")
	h := NewAPIHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	h.health(res, req)
	if res.Code != http.StatusOK { t.Fatalf("status = %d, want %d", res.Code, http.StatusOK) }
	if !strings.Contains(res.Body.String(), "healthy") { t.Fatalf("unexpected response: %s", res.Body.String()) }
}


func TestUpdateAndDeleteTodo(t *testing.T) {
	repo := newFakeRepository()
	h := NewAPIHandler(repo)
	created, err := repo.Create(context.Background(), Todo{Title: "Original"})
	if err != nil { t.Fatal(err) }

	updateReq := httptest.NewRequest(http.MethodPut, "/todos/"+created.ID.Hex(), strings.NewReader(`{"title":"Updated","completed":true}`))
	updateReq.SetPathValue("id", created.ID.Hex())
	updateRes := httptest.NewRecorder()
	h.updateTodo(updateRes, updateReq)
	if updateRes.Code != http.StatusOK { t.Fatalf("update status = %d, body=%s", updateRes.Code, updateRes.Body.String()) }
	updated, err := repo.Get(context.Background(), created.ID)
	if err != nil { t.Fatal(err) }
	if updated.Title != "Updated" || !updated.Completed { t.Fatalf("unexpected updated todo: %+v", updated) }

	deleteReq := httptest.NewRequest(http.MethodDelete, "/todos/"+created.ID.Hex(), nil)
	deleteReq.SetPathValue("id", created.ID.Hex())
	deleteRes := httptest.NewRecorder()
	h.deleteTodo(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent { t.Fatalf("delete status = %d, want %d", deleteRes.Code, http.StatusNoContent) }
	if _, err := repo.Get(context.Background(), created.ID); !errors.Is(err, ErrTodoNotFound) { t.Fatalf("Get error = %v, want ErrTodoNotFound", err) }
}

func TestRejectsUnknownJSONFields(t *testing.T) {
	h := NewAPIHandler(newFakeRepository())
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"Valid","unexpected":true}`))
	res := httptest.NewRecorder()
	h.createTodo(res, req)
	if res.Code != http.StatusBadRequest { t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest) }
}

func TestRejectsOversizedTitle(t *testing.T) {
	h := NewAPIHandler(newFakeRepository())
	title := strings.Repeat("x", 201)
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"`+title+`"}`))
	res := httptest.NewRecorder()
	h.createTodo(res, req)
	if res.Code != http.StatusBadRequest { t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest) }
}
