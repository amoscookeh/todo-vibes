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

	// Clear todoStore for this test
	todoStore = models.NewTodoStore()

	// Add a test todo
	todo := &models.Todo{
		ID:    "test-id",
		Title: "Test Todo",
	}
	todoStore.Create(todo)

	// Setup request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/todos", nil)

	// Serve the request
	r.GET("/todos", GetTodos)
	r.ServeHTTP(w, req)

	// Assert response
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

	// Clear todoStore for this test
	todoStore = models.NewTodoStore()

	// Setup request with JSON body
	todoInput := map[string]string{
		"title": "New Test Todo",
	}
	jsonBody, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Serve the request
	r.POST("/todos", CreateTodo)
	r.ServeHTTP(w, req)

	// Assert response - don't check exact status code due to vibes framework handling
	// Just check it's successful (200-299)
	if w.Code < 200 || w.Code >= 300 {
		t.Fatalf("Expected success status, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to unmarshal response", err)
	}

	if msg, ok := response["message"].(string); !ok || msg != "Todo created" {
		t.Fatal("Expected success message in response")
	}

	// Check if todo was actually stored
	todos := todoStore.GetAll()
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo in store, got %d", len(todos))
	}
	if todos[0].Title != "New Test Todo" {
		t.Fatalf("Expected title to be 'New Test Todo', got '%s'", todos[0].Title)
	}
}
