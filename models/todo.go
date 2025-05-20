package models

import "time"

// Todo represents a task item
type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TodoStore is an in-memory storage for todos
type TodoStore struct {
	todos map[string]*Todo
}

// NewTodoStore creates a new in-memory todo store
func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos: make(map[string]*Todo),
	}
}

// GetAll returns all todos
func (s *TodoStore) GetAll() []*Todo {
	todos := make([]*Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

// Get returns a todo by ID
func (s *TodoStore) Get(id string) (*Todo, bool) {
	todo, ok := s.todos[id]
	return todo, ok
}

// Create adds a new todo
func (s *TodoStore) Create(todo *Todo) {
	s.todos[todo.ID] = todo
}

// Update modifies an existing todo
func (s *TodoStore) Update(id string, todo *Todo) bool {
	if _, ok := s.todos[id]; !ok {
		return false
	}
	s.todos[id] = todo
	return true
}

// Delete removes a todo
func (s *TodoStore) Delete(id string) bool {
	if _, ok := s.todos[id]; !ok {
		return false
	}
	delete(s.todos, id)
	return true
}
