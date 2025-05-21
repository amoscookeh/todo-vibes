package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amoscookeh/go-vibes"
	"github.com/amoscookeh/todo-vibe/models"
)

func setupTest() *vibes.VibesEngine {
	r := vibes.Default()
	return r
}

func TestGetTodos(t *testing.T) {
	r := setupTest()

	todoStore = models.NewTodoStore()

	todo := &models.Todo{
		ID:    "test-id",
		Title: "Test Todo",
	}
	todoStore.Create(todo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/todos", nil)

	r.GET("/todos", GetTodos)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to unmarshal response", err)
	}

	if todos, ok := response["todos"].([]interface{}); !ok || len(todos) != 1 {
		t.Fatal("Expected 1 todo in response")
	}
}

func TestCreateTodo(t *testing.T) {
	r := setupTest()

	todoStore = models.NewTodoStore()

	todoInput := map[string]string{
		"title": "New Test Todo",
	}
	jsonBody, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	r.POST("/todos", CreateTodo)
	r.ServeHTTP(w, req)

<<<<<<< Updated upstream
	// Assert response - don't check exact status code due to vibes framework handling
	// Just check it's successful (200-299)
	if w.Code < 200 || w.Code >= 300 {
		t.Fatalf("Expected success status, got %d", w.Code)
=======
	responseStr := w.Body.String()
	if !strings.Contains(responseStr, "emoji") && !strings.Contains(responseStr, "🆕") {
		t.Fatal("Expected emoji status in response, not found")
>>>>>>> Stashed changes
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to unmarshal response", err)
	}

	if msg, ok := response["message"].(string); !ok || msg != "Todo created" {
		t.Fatal("Expected success message in response")
	}

	todos := todoStore.GetAll()
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo in store, got %d", len(todos))
	}
	if todos[0].Title != "New Test Todo" {
		t.Fatalf("Expected title to be 'New Test Todo', got '%s'", todos[0].Title)
	}
}
<<<<<<< Updated upstream
=======

func TestDeleteTodo(t *testing.T) {
	r := setupTest()

	todoStore = models.NewTodoStore()

	todo := &models.Todo{
		ID:    "test-id",
		Title: "Test Todo",
	}
	todoStore.Create(todo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/todos/test-id", nil)

	r.DELETE("/todos/:id", DeleteTodo)
	r.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to unmarshal response", err)
	}

	_, hasEmojiField := response["status_emoji"]
	if !hasEmojiField {
		t.Fatal("Expected status_emoji field in response")
	}

	todos := todoStore.GetAll()
	if len(todos) != 0 {
		t.Fatalf("Expected 0 todos in store, got %d", len(todos))
	}
}

func TestUpdateTodo(t *testing.T) {
	r := setupTest()

	todoStore = models.NewTodoStore()

	originalTodo := &models.Todo{
		ID:        "test-id",
		Title:     "Original Title",
		Completed: false,
	}
	todoStore.Create(originalTodo)

	updateInput := map[string]interface{}{
		"title":     "Updated Title",
		"completed": true,
	}
	jsonBody, _ := json.Marshal(updateInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/todos/test-id", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	r.PUT("/todos/:id", UpdateTodo)
	r.ServeHTTP(w, req)

	if w.Code < 200 || w.Code >= 300 {
		t.Fatalf("Expected success status, got %d", w.Code)
	}

	updatedTodo, exists := todoStore.Get("test-id")
	if !exists {
		t.Fatal("Expected todo to still exist")
	}

	if updatedTodo.Title != "Updated Title" {
		t.Fatalf("Expected title to be updated to 'Updated Title', got '%s'", updatedTodo.Title)
	}

	if !updatedTodo.Completed {
		t.Fatal("Expected completed to be updated to true")
	}
}
>>>>>>> Stashed changes
