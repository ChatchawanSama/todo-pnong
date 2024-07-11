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
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func getTodo(ctx *gin.Context) {
	fmt.Println("Entering getTodo handler")
	todos := []Todo{}

	rows, err := database.DB.Query("SELECT id, title, done FROM todos")
	if err != nil {
		log.Fatal("can't query all todos", err)
	}

	for rows.Next() {
		var id int
		var title string
		var done bool
		err := rows.Scan(&id, &title, &done)
		if err != nil {
			log.Fatal("can't Scan row into variable", err)
		}
		fmt.Println(id, title, done)
		todos = append(todos, Todo{id, title, done})
	}

	fmt.Println("query all todos success")
	ctx.JSON(http.StatusOK, todos)

}

func getTodoByID(ctx *gin.Context) {
	rowId := ctx.Param("id")

	q := "SELECT id, title, done FROM todos where id=$1"
	row := database.DB.QueryRow(q, rowId)
	var id int
	var title string
	var done bool

	err := row.Scan(&id, &title, &done)
	if err != nil {
		log.Fatal("can't Scan row into variables", err)
	}

	fmt.Println("one row", id, title, done)
	ctx.JSON(http.StatusOK, Todo{id, title, done})
}

func postTodo(ctx *gin.Context) {
	fmt.Println("Entering postTodo handler")
	var todo Todo

	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "INSERT INTO todos (title, done) VALUES ($1, $2) RETURNING id"
	err := database.DB.QueryRow(query, todo.Title, todo.Done).Scan(&todo.ID)
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

	// var title string
	// var done bool
	var todo Todo

	if err := ctx.BindJSON(&todo); err != nil {
		fmt.Println("Error binding JSON:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "UPDATE todos SET title=$2, done=$3 WHERE id=$1 RETURNING id;"
	if _, err := database.DB.Exec(query, rowId, todo.Title, todo.Done); err != nil {
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
	ctx.JSON(http.StatusCreated, Todo{idInt, todo.Title, todo.Done})
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
