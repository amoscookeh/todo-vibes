# Todo Vibe

A simple todo API built with the Go Vibes framework.

## Setup

1. Clone the repository
2. Install dependencies:
```bash
go mod tidy
```

## Run the application

```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## API Endpoints

- **GET /todos**: Get all todos
- **GET /todos/:id**: Get a specific todo
- **POST /todos**: Create a new todo
- **PUT /todos/:id**: Update a todo
- **DELETE /todos/:id**: Delete a todo

## Example Requests

### Create a todo

```bash
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go Vibes"}'
```

### Get all todos

```bash
curl http://localhost:8080/todos
```

### Update a todo

```bash
curl -X PUT http://localhost:8080/todos/{id} \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'
```

### Delete a todo

```bash
curl -X DELETE http://localhost:8080/todos/{id}
``` 