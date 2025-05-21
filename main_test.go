package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amoscookeh/go-vibes"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *vibes.VibesEngine {
	r := vibes.Default()
	initRoutes(r)
	return r
}

func TestCreateTodo(t *testing.T) {
	router := setupTestRouter()

	todoInput := map[string]interface{}{
		"title": "Test Todo",
	}
	jsonData, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Todo created", response["message"])

	todo := response["todo"].(map[string]interface{})
	assert.Equal(t, "Test Todo", todo["title"])
	assert.Equal(t, false, todo["completed"])
	assert.NotEmpty(t, todo["id"])
}

func TestGetTodos(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/todos", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Contains(t, response, "todos")
}

func TestGetSingleTodo(t *testing.T) {
	router := setupTestRouter()

	todoInput := map[string]interface{}{
		"title": "Get Single Test",
	}
	jsonData, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	todoID := createResponse["todo"].(map[string]interface{})["id"].(string)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/todos/"+todoID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Contains(t, response, "todo")
	todo := response["todo"].(map[string]interface{})
	assert.Equal(t, "Get Single Test", todo["title"])
	assert.Equal(t, todoID, todo["id"])
}

func TestUpdateTodo(t *testing.T) {
	router := setupTestRouter()

	todoInput := map[string]interface{}{
		"title": "Update Test Todo",
	}
	jsonData, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	todoID := createResponse["todo"].(map[string]interface{})["id"].(string)

	updateInput := map[string]interface{}{
		"title":     "Updated Todo",
		"completed": true,
	}
	jsonData, _ = json.Marshal(updateInput)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/todos/"+todoID, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Todo updated", response["message"])

	todo := response["todo"].(map[string]interface{})
	assert.Equal(t, "Updated Todo", todo["title"])
	assert.Equal(t, true, todo["completed"])
}

func TestDeleteTodo(t *testing.T) {
	router := setupTestRouter()

	todoInput := map[string]interface{}{
		"title": "Delete Test Todo",
	}
	jsonData, _ := json.Marshal(todoInput)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	todoID := createResponse["todo"].(map[string]interface{})["id"].(string)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/todos/"+todoID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Todo deleted", response["message"])

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/todos/"+todoID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.Contains(t, getResponse, "message")
	assert.Equal(t, "Todo not found", getResponse["message"])
}
