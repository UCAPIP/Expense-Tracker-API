package main

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"
)

const schemaExpenses = `
CREATE TABLE IF NOT EXISTS Expenses (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    description   TEXT NOT NULL,
	amount        INTEGER NOT NULL,
	category_id   INTEGER NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
	user_id       INTEGER NOT NULL
);`

const schemaUsers = `
CREATE TABLE IF NOT EXISTS Users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password      TEXT NOT NULL
);`

const schemaCategories = `
CREATE TABLE IF NOT EXISTS Categories (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
	category_name INTEGER NOT NULL,
	user_id       INTEGER NOT NULL
);`

func initDB(db *sql.DB) error {
	if _, err := db.Exec(schemaUsers); err != nil {
		return err
	}
	if _, err := db.Exec(schemaExpenses); err != nil {
		return err
	}
	if _, err := db.Exec(schemaCategories); err != nil {
		return err
	}
	return nil
}

type Expenses struct {
	Id          int    `json:"id"`
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	CategoryId  int    `json:"categoryId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	UserId      int    `json:"userId"`
}

type User struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password"`
}

type Category struct {
	Id           int    `json:"id"`
	CategoryName string `json:"categoryName"`
	UserId       int    `json:"userId"`
}

func loading(db *sql.DB, userId int) ([]Expenses, error) {

	var exp = []Expenses{}

	query := "SELECT * FROM Tasks WHERE user_id = ? ORDER BY id"

	rows, err := db.Query(query, userId)
	if err != nil {
		return exp, err
	}
	defer rows.Close()

	for rows.Next() {
		var e Expenses
		if err := rows.Scan(&e.Id, &e.Description, &e.Amount, &e.CategoryId, &e.CreatedAt, &e.UpdatedAt, &e.UserId); err != nil {
			return exp, err
		}

		exp = append(exp, e)
	}

	if err := rows.Err(); err != nil {
		return exp, err
	}

	return exp, nil
}

func finding(db *sql.DB, taskId, userId int) (*Expenses, error) {
	var e Expenses

	err := db.QueryRow(
		"SELECT id, title, description, created_at, updated_at, user_id FROM Tasks WHERE id = ? AND user_id = ?",
		taskId, userId,
	).Scan(&e.Id, &e.Description, &e.Amount, &e.CategoryId, &e.CreatedAt, &e.UpdatedAt, &e.UserId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func findingCategory(db *sql.DB, category *string, userId int) (bool, error) {
	var c Category

	normalizedCategory := strings.ToLower(*category)

	err := db.QueryRow(
		"SELECT * FROM Categories WHERE category_name = ? AND user_id = ?",
		normalizedCategory, userId,
	).Scan(&c.Id, &c.CategoryName, &c.UserId)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func addingCategory(db *sql.DB, category *string, userId int) error {

	normalizedCategory := strings.ToLower(*category)

	_, err := db.Exec("INSERT INTO Categories (category_name, user_id) VALUES (?, ?)",
		normalizedCategory, userId,
	)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func adding(db *sql.DB, input *ExpensesInput, userId int) (int64, error) {

	now := time.Now().Format(time.RFC3339)

	res, err := db.Exec("INSERT INTO Expenses (title, amount, category_id, created_at, updated_at, user_id) VALUES (?, ?, ?, ?, ?)",
		input.Title, input.Amount, now, now, userId,
	)

	if err != nil {
		log.Println(err)
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Println(err)
		return 0, err
	}
	log.Println(id, input.Title, "ID Expenses was created succesfuly")
	return id, nil

}

func editing(db *sql.DB, input *ExpensesInput, id, userId int) error {
	now := time.Now().Format(time.RFC3339)

	res, err := db.Exec("UPDATE Tasks SET title = ?, description = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		input.Title, input.Amount, now, id, userId,
	)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func deleting(db *sql.DB, id, userId int) error {
	res, err := db.Exec("DELETE FROM Tasks WHERE id = ? AND user_id = ?", id, userId)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func createUser(db *sql.DB, name, email, passHash string) (int64, error) {
	res, err := db.Exec("INSERT INTO Users (name, email, password) VALUES (?, ?, ?)",
		name, email, passHash,
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, ErrUserExists
		}
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Println(err)
		return 0, err
	}

	log.Println(id, "User was registrated succesfuly")
	return id, nil
}

func findUserByUsername(db *sql.DB, email string) (*User, error) {
	var targetUser = User{}

	err := db.QueryRow(
		"SELECT id, name, email, password FROM Users WHERE email = ?",
		email,
	).Scan(&targetUser.Id, &targetUser.Name, &targetUser.Email, &targetUser.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &targetUser, nil
}

func taskOwner(db *sql.DB, taskID int) (int, error) {
	var owner int
	err := db.QueryRow("SELECT user_id FROM Tasks WHERE id = ?", taskID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return owner, err
}
