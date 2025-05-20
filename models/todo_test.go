package models

import (
	"testing"
	"time"
)

func TestNewTodoStore(t *testing.T) {
	store := NewTodoStore()
	if store == nil {
		t.Fatal("Expected store to be initialized, got nil")
	}
	if store.todos == nil {
		t.Fatal("Expected todos map to be initialized, got nil")
	}
}

func TestTodoStore_Create(t *testing.T) {
	store := NewTodoStore()
	todo := &Todo{
		ID:        "test-id",
		Title:     "Test Todo",
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	store.Create(todo)

	if len(store.todos) != 1 {
		t.Fatalf("Expected 1 todo in store, got %d", len(store.todos))
	}

	if stored, ok := store.todos["test-id"]; !ok || stored != todo {
		t.Fatal("Todo was not stored correctly")
	}
}

func TestTodoStore_Get(t *testing.T) {
	store := NewTodoStore()
	todo := &Todo{
		ID:        "test-id",
		Title:     "Test Todo",
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	store.todos["test-id"] = todo

	// Test existing todo
	found, exists := store.Get("test-id")
	if !exists {
		t.Fatal("Expected todo to exist")
	}
	if found != todo {
		t.Fatal("Expected to get the same todo that was stored")
	}

	// Test non-existing todo
	_, exists = store.Get("non-existing")
	if exists {
		t.Fatal("Expected todo to not exist")
	}
}

func TestTodoStore_GetAll(t *testing.T) {
	store := NewTodoStore()
	todo1 := &Todo{ID: "1", Title: "Todo 1"}
	todo2 := &Todo{ID: "2", Title: "Todo 2"}

	store.todos["1"] = todo1
	store.todos["2"] = todo2

	todos := store.GetAll()

	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(todos))
	}
}

func TestTodoStore_Update(t *testing.T) {
	store := NewTodoStore()
	todo := &Todo{
		ID:        "test-id",
		Title:     "Test Todo",
		Completed: false,
	}

	store.todos["test-id"] = todo

	updatedTodo := &Todo{
		ID:        "test-id",
		Title:     "Updated Todo",
		Completed: true,
	}

	// Test successful update
	success := store.Update("test-id", updatedTodo)
	if !success {
		t.Fatal("Expected update to succeed")
	}
	if store.todos["test-id"].Title != "Updated Todo" {
		t.Fatal("Expected title to be updated")
	}

	// Test update of non-existing todo
	success = store.Update("non-existing", updatedTodo)
	if success {
		t.Fatal("Expected update to fail for non-existing todo")
	}
}

func TestTodoStore_Delete(t *testing.T) {
	store := NewTodoStore()
	todo := &Todo{ID: "test-id", Title: "Test Todo"}

	store.todos["test-id"] = todo

	// Test successful delete
	success := store.Delete("test-id")
	if !success {
		t.Fatal("Expected delete to succeed")
	}
	if len(store.todos) != 0 {
		t.Fatal("Expected store to be empty after delete")
	}

	// Test delete of non-existing todo
	success = store.Delete("non-existing")
	if success {
		t.Fatal("Expected delete to fail for non-existing todo")
	}
}
