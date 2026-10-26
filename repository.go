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
	spent_on      TEXT NOT NULL,
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
	category_name INTEGER NOT NULL
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
	return seedCategories(db)
}

var defaultCategories = []string{"Groceries", "Leisure", "Electronics", "Utilities", "Clothing", "Health", "Others"}

func seedCategories(db *sql.DB) error {
	for _, name := range defaultCategories {
		if _, err := db.Exec("INSERT OR IGNORE INTO Categories (category_name) VALUES (?)", name); err != nil {
			return err
		}
	}
	return nil
}

type Expenses struct {
	Id          int    `json:"id"`
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	CategoryId  int    `json:"categoryId"`
	SpentOn     string `json:"spentOn"`
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
}

func loading(db *sql.DB, userId int, dr DateRange) ([]Expenses, error) {
	query := `SELECT id, description, amount, category_id, spent_on, created_at, updated_at, user_id
	          FROM Expenses WHERE user_id = ?`
	args := []any{userId}

	if dr.From != "" {
		query += " AND spent_on >= ?"
		args = append(args, dr.From)
	}
	if dr.To != "" {
		query += " AND spent_on <= ?"
		args = append(args, dr.To)
	}
	query += " ORDER BY spent_on DESC, id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exp := []Expenses{}
	for rows.Next() {
		var e Expenses
		if err := rows.Scan(&e.Id, &e.Description, &e.Amount, &e.CategoryId, &e.SpentOn, &e.CreatedAt, &e.UpdatedAt, &e.UserId); err != nil {
			return nil, err
		}
		exp = append(exp, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return exp, nil
}

func finding(db *sql.DB, expenseId, userId int) (*Expenses, error) {
	var e Expenses

	err := db.QueryRow(
		`SELECT id, description, amount, category_id, spent_on, created_at, updated_at, user_id
		 FROM Expenses WHERE id = ? AND user_id = ?`,
		expenseId, userId,
	).Scan(&e.Id, &e.Description, &e.Amount, &e.CategoryId, &e.SpentOn, &e.CreatedAt, &e.UpdatedAt, &e.UserId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func findingCategory(db *sql.DB, category string) (int, error) {
	var id int
	err := db.QueryRow(
		"SELECT id FROM Categories WHERE category_name = ? COLLATE NOCASE",
		strings.TrimSpace(category),
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

//func addingCategory(db *sql.DB, category *string, userId int) error {
//
//	normalizedCategory := strings.ToLower(*category)
//
//	_, err := db.Exec("INSERT INTO Categories (category_name, user_id) VALUES (?, ?)",
//		normalizedCategory, userId,
//	)
//
//	if err != nil {
//		log.Println(err)
//		return err
//	}
//
//	return nil
//}

func adding(db *sql.DB, input *ExpensesInput, userId, categoryID int) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := db.Exec(
		`INSERT INTO Expenses (description, amount, category_id, spent_on, created_at, updated_at, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Description, input.Amount, categoryID, input.SpentOn, now, now, userId,
	)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func editing(db *sql.DB, input *ExpensesInput, id, userId, categoryId int) error {
	now := time.Now().Format(time.RFC3339)

	res, err := db.Exec("UPDATE Expenses SET description = ?, amount = ?, spent_on = ?, updated_at = ?, category_id = ? WHERE id = ? AND user_id = ?",
		input.Description, input.Amount, input.SpentOn, now, categoryId, id, userId,
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
	res, err := db.Exec("DELETE FROM Expenses WHERE id = ? AND user_id = ?", id, userId)
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

func taskOwner(db *sql.DB, expensesId int) (int, error) {
	var owner int
	err := db.QueryRow("SELECT user_id FROM Expenses WHERE id = ?", expensesId).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return owner, err
}
