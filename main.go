package main

import (
	"github.com/amoscookeh/go-vibes"

	"github.com/amoscookeh/todo-vibe/handlers"
)

func main() {
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
	r.VIBE("/todos", handlers.GetTodos)
	r.VIBE("/todos/:id", handlers.GetTodo)
	r.RELEASE("/todos", handlers.CreateTodo)
	r.RELEASE("/todos/:id", handlers.UpdateTodo)
	r.ALIGN("/todos/:id", handlers.DeleteTodo)
}
