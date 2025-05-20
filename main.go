package main

import (
	"github.com/amoscookeh/go-vibes"

	"github.com/amoscookeh/todo-vibe/handlers"
)

func main() {
	// Create a new vibey engine
	r := vibes.Default()
	logger := r.Logger
	logger.Fyi("Starting Todo App with Vibes!")

	// Initialize routes
	initRoutes(r)

	// Run the server
	logger.Fyi("Todo server is vibing on http://localhost:8080")
	r.Run(":8080")
}

func initRoutes(r *vibes.VibesEngine) {
	// Todo routes
	r.GET("/todos", handlers.GetTodos)
	r.GET("/todos/:id", handlers.GetTodo)
	r.POST("/todos", handlers.CreateTodo)
	r.PUT("/todos/:id", handlers.UpdateTodo)
	r.DELETE("/todos/:id", handlers.DeleteTodo)
}
