package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

type App struct {
	db        *sql.DB
	jwtSecret []byte
}

func main() {
	db, err := sql.Open("sqlite", "store.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system environment variables")
	}

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters long")
	}
	app := &App{db: db, jwtSecret: []byte(secret)}

	http.HandleFunc("POST /register", app.userRegister)
	http.HandleFunc("POST /login", app.userLogin)

	http.HandleFunc("GET /expenses", app.requireAuth(app.getExpenses))
	http.HandleFunc("POST /expenses", app.requireAuth(app.addNewExpenses))
	http.HandleFunc("DELETE /tasks/{id}", app.requireAuth(app.deleteExpensesById))
	http.HandleFunc("PUT /tasks/{id}", app.requireAuth(app.updateExpensesById))

	log.Println("The server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
