package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amoscookeh/go-vibes"
	"github.com/amoscookeh/todo-vibe/models"
)

func TestMiddlewareDeleteTodo(t *testing.T) {
	r := vibes.Default()

	todoStore = models.NewTodoStore()

	todo := &models.Todo{
		ID:    "test-id",
		Title: "Test Todo",
	}
	todoStore.Create(todo)

	r.DELETE("/todos/:id", DeleteTodo)

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", ts.URL+"/todos/test-id", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Failed to send request:", err)
	}
	defer resp.Body.Close()

	// Check response body for status_emoji instead of header
	bodyBytes, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatal("Failed to unmarshal response", err)
	}

	_, hasEmojiField := response["status_emoji"]
	if !hasEmojiField {
		t.Fatal("Missing status_emoji field in response")
	}
}
