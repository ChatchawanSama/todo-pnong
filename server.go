package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"todo-pnong/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Todo struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func getTodo(ctx *gin.Context) {
	fmt.Println("Entering getTodo handler")
	todos := []Todo{}

	rows, err := database.DB.Query("SELECT id, title, status FROM todos")
	if err != nil {
		log.Fatal("can't query all todos", err)
	}

	for rows.Next() {
		var id int
		var title, status string
		err := rows.Scan(&id, &title, &status)
		if err != nil {
			log.Fatal("can't Scan row into variable", err)
		}
		fmt.Println(id, title, status)
		todos = append(todos, Todo{id, title, status})
	}

	fmt.Println("query all todos success")
	// ctx.JSON(http.StatusOK, todos)
	ctx.JSON(http.StatusOK, gin.H{"data": todos})

}

func getTodoByID(ctx *gin.Context) {
	rowId := ctx.Param("id")

	q := "SELECT id, title, status FROM todos where id=$1"
	row := database.DB.QueryRow(q, rowId)
	var id int
	var title, status string

	err := row.Scan(&id, &title, &status)
	if err != nil {
		log.Fatal("can't Scan row into variables", err)
	}

	fmt.Println("one row", id, title, status)
	// ctx.JSON(http.StatusOK, Todo{id, title, status})
	ctx.JSON(http.StatusOK, gin.H{"data": Todo{id, title, status}})
}

func postTodo(ctx *gin.Context) {
	fmt.Println("Entering postTodo handler")
	var todo Todo

	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "INSERT INTO todos (title, status) VALUES ($1, $2) RETURNING id"
	err := database.DB.QueryRow(query, todo.Title, todo.Status).Scan(&todo.ID)
	if err != nil {
		fmt.Println("Error inserting new todo:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("Todo created with ID:", todo.ID)
	ctx.JSON(http.StatusCreated, todo)
}

func putTodoByID(ctx *gin.Context) {
	fmt.Println("Entering putTodoByID handler")

	rowId := ctx.Param("id")

	var todo Todo

	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "UPDATE todos SET title=$2, status=$3 WHERE id=$1 RETURNING id;"
	if _, err := database.DB.Exec(query, rowId, todo.Title, todo.Status); err != nil {
		fmt.Println("Error executing update:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	idInt, err := strconv.Atoi(rowId)

	if err != nil {
		fmt.Println("Error during conversion")
		return
	}

	fmt.Println("update success")
	ctx.JSON(http.StatusCreated, Todo{idInt, todo.Title, todo.Status})
}

func deleteTodoByID(ctx *gin.Context) {
	fmt.Println("Entering deleteTodoByID handler")

	rowId := ctx.Param("id")

	query := "DELETE FROM todos WHERE id=$1 RETURNING id;"
	result, err := database.DB.Exec(query, rowId)
	if err != nil {
		fmt.Println("Error executing delete:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Error fetching affected rows:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		return
	}

	fmt.Println("Delete success")
	ctx.JSON(http.StatusOK, gin.H{"response": "success"})
}

func patchTodoStatusByID(ctx *gin.Context) {
	fmt.Println("Entering patchTodoStatusByID handler")

	var todo Todo

	rowId := ctx.Param("id")
	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := database.DB.Exec("UPDATE todos SET status=$2 WHERE id=$1;", rowId, todo.Status); err != nil {
		log.Fatal("error execute update ", err)
	}

	fmt.Println("update success")
	ctx.JSON(http.StatusOK, todo)
}

func patchTodoTitleByID(ctx *gin.Context) {
	fmt.Println("Entering patchTodoTitleByID handler")

	var todo Todo

	rowId := ctx.Param("id")
	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := database.DB.Exec("UPDATE todos SET title=$2 WHERE id=$1;", rowId, todo.Title); err != nil {
		log.Fatal("error execute update ", err)
	}

	fmt.Println("update success")
	ctx.JSON(http.StatusOK, todo)
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Connect to the database
	database.ConnectDB()
	defer database.DB.Close()

	// Create tables
	database.CreateTable()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	r := gin.Default()
	r.GET("/api/v1/todos", getTodo)
	r.GET("/api/v1/todos/:id", getTodoByID)
	r.POST("/api/v1/todos", postTodo)
	r.PUT("/api/v1/todos/:id", putTodoByID)
	r.DELETE("/api/v1/todos/:id", deleteTodoByID)
	r.PATCH("/api/v1/todos/:id/actions/status", patchTodoStatusByID)
	r.PATCH("/api/v1/todos/:id/actions/title", patchTodoTitleByID)

	port := os.Getenv("HOST")
	// if port == "" {
	// 	fmt.Println("Why Port Is String?!!")
	// 	port = "8080" // default port if not specified
	// }

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	serverErrors := make(chan error, 1)

	// Start the service listening for requests
	go func() {
		log.Printf("Listening on port %s", port)
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("Received shutdown signal, gracefully shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}

	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Error starting server: %v", err)
		}
	}

	log.Println("Server stopped")
}
