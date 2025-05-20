package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amoscookeh/go-vibes"
)

func TestInitRoutes(t *testing.T) {
	r := vibes.Default()

	// Initialize routes
	initRoutes(r)

	// Check if routes are registered by making test requests
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/todos"},
		{"GET", "/todos/123"},
		{"POST", "/todos"},
		{"PUT", "/todos/123"},
		{"DELETE", "/todos/123"},
	}

	for _, route := range routes {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(route.method, route.path, nil)
		r.ServeHTTP(w, req)

		// We don't care about the response code here, just that the route exists
		// and doesn't trigger a 404 (which would be a 404 emoji in vibes)
		if w.Code == http.StatusNotFound {
			t.Fatalf("Route %s %s not found", route.method, route.path)
		}
	}
}
