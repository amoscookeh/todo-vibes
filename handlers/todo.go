package handlers

import (
	"net/http"
	"time"

	"github.com/amoscookeh/go-vibes"
	"github.com/google/uuid"

	"github.com/amoscookeh/todo-vibe/models"
)

// Global store for todos
var todoStore = models.NewTodoStore()

// GetTodos returns all todos
func GetTodos(c *vibes.Context) {
	todos := todoStore.GetAll()
	c.JSON(http.StatusOK, vibes.Map{
		"todos": todos,
	})
}

// GetTodo returns a specific todo by ID
func GetTodo(c *vibes.Context) {
	id := c.Param("id")
	todo, exists := todoStore.Get(id)

	if !exists {
		c.JSON(http.StatusNotFound, vibes.Map{
			"message": "Todo not found",
		})
		return
	}

	c.JSON(http.StatusOK, vibes.Map{
		"todo": todo,
	})
}

// CreateTodo adds a new todo
func CreateTodo(c *vibes.Context) {
	var input struct {
		Title string `json:"title"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, vibes.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
		return
	}

	now := time.Now()
	todo := &models.Todo{
		ID:        uuid.New().String(),
		Title:     input.Title,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	todoStore.Create(todo)

	c.JSON(http.StatusCreated, vibes.Map{
		"message": "Todo created",
		"todo":    todo,
	})
}

// UpdateTodo modifies an existing todo
func UpdateTodo(c *vibes.Context) {
	id := c.Param("id")

	var input struct {
		Title     *string `json:"title"`
		Completed *bool   `json:"completed"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, vibes.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
		return
	}

	todo, exists := todoStore.Get(id)
	if !exists {
		c.JSON(http.StatusNotFound, vibes.Map{
			"message": "Todo not found",
		})
		return
	}

	if input.Title != nil {
		todo.Title = *input.Title
	}

	if input.Completed != nil {
		todo.Completed = *input.Completed
	}

	todo.UpdatedAt = time.Now()

	if todoStore.Update(id, todo) {
		c.JSON(http.StatusOK, vibes.Map{
			"message": "Todo updated",
			"todo":    todo,
		})
	} else {
		c.JSON(http.StatusInternalServerError, vibes.Map{
			"message": "Failed to update todo",
		})
	}
}

// DeleteTodo removes a todo
func DeleteTodo(c *vibes.Context) {
	id := c.Param("id")

	if !todoStore.Delete(id) {
		c.JSON(http.StatusNotFound, vibes.Map{
			"message": "Todo not found",
		})
		return
	}

	c.JSON(http.StatusOK, vibes.Map{
		"message": "Todo deleted",
	})
}
