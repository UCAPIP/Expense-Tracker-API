package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (app *App) getExpenses(w http.ResponseWriter, r *http.Request, userId int) {
	dr, err := parseRange(r.URL.Query(), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	expenses, err := loading(app.db, userId, dr)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(w, http.StatusOK, expenses)
}

type ExpensesInput struct {
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	Category    string `json:"category"`
	SpentOn     string `json:"spentOn"`
}

func validate(input *ExpensesInput) []string {
	var problems []string
	if strings.TrimSpace(input.Description) == "" {
		problems = append(problems, "Description is required")
	}
	if input.Amount <= 0 {
		problems = append(problems, "Amount must be greater than 0")
	}
	if strings.TrimSpace(input.Category) == "" {
		problems = append(problems, "Category is required")
	}
	return problems
}

func normalizeSpentOn(input *ExpensesInput) error {
	if input.SpentOn == "" {
		input.SpentOn = time.Now().UTC().Format(dateLayout)
		return nil
	}
	if _, err := time.Parse(dateLayout, input.SpentOn); err != nil {
		return errors.New("spentOn must be in YYYY-MM-DD format")
	}
	return nil
}

func (app *App) addNewExpenses(w http.ResponseWriter, r *http.Request, userId int) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input ExpensesInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "")
		return
	}

	if problems := validate(&input); len(problems) > 0 {
		writeError(w, http.StatusBadRequest, "Validation failed", strings.Join(problems, "; "))
		return
	}

	if err := normalizeSpentOn(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	if input.SpentOn == "" {
		input.SpentOn = time.Now().UTC().Format(dateLayout) // не прислал дату, значит сегодня
	} else if _, err := time.Parse(dateLayout, input.SpentOn); err != nil {
		writeError(w, http.StatusBadRequest, "Validation failed", "date must be in YYYY-MM-DD format")
		return
	}

	categoryId, err := findingCategory(app.db, input.Category)
	if err != nil {
		log.Println(err, input.Category, "Category doesn't exist")
		writeError(w, http.StatusNotFound, "Category not found", "")
		return
	}
	//if !categoryIsExist {
	//	err := addingCategory(app.db, &input.CategoryId, userId)
	//	if err != nil {
	//		log.Println(err)
	//		writeError(w, http.StatusInternalServerError, "Internal server error", "An unexpected error occurred on the server. Please try again later.")
	//		return
	//	}
	//}

	exnpensesId, err := adding(app.db, &input, userId, categoryId)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "An unexpected error occurred on the server. Please try again later.")
		return
	}

	newExpenses, err := finding(app.db, int(exnpensesId), userId)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(w, http.StatusCreated, newExpenses)
}

func (app *App) updateExpensesById(w http.ResponseWriter, r *http.Request, userId int) {
	expensesIdStr := r.PathValue("id")
	expensesId, err := strconv.Atoi(expensesIdStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request", "The 'id' parameter must be a valid integer number.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var input ExpensesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "")
		return
	}

	if problems := validate(&input); len(problems) > 0 {
		writeError(w, http.StatusBadRequest, "Validation failed", strings.Join(problems, "; "))
		return
	}

	if err := normalizeSpentOn(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	owner, err := taskOwner(app.db, expensesId)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Expenses not found", "")
		return
	}
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}
	if owner != userId {
		writeError(w, http.StatusForbidden, "Forbidden", "")
		return
	}

	categoryId, err := findingCategory(app.db, input.Category)
	if err != nil {
		log.Println(err, input.Category, "Category doesn't exist")
		writeError(w, http.StatusNotFound, "Category not found", "")
		return
	}

	err = editing(app.db, &input, expensesId, userId, categoryId)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Expenses not found", "")
		return
	}
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	updatedTask, err := finding(app.db, expensesId, userId)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	writeJSON(w, http.StatusOK, updatedTask)

}

func (app *App) deleteExpensesById(w http.ResponseWriter, r *http.Request, userId int) {
	expensesIdStr := r.PathValue("id")
	expensesId, err := strconv.Atoi(expensesIdStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request", "The 'id' parameter must be a valid integer number.")
		return
	}

	owner, err := taskOwner(app.db, expensesId)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Task not found", "")
		return
	}
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}
	if owner != userId {
		writeError(w, http.StatusForbidden, "Forbidden", "")
		return
	}

	err = deleting(app.db, expensesId, userId)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Task not found", "")
		return
	}
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type Credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *App) userRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var c Credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "")
		return
	}

	c.Name = strings.TrimSpace(c.Name)
	c.Email = strings.TrimSpace(c.Email)
	if c.Email == "" || c.Name == "" || len(c.Password) < 8 || len(c.Password) > 72 {
		writeError(w, http.StatusBadRequest, "Validation failed",
			"Name is required; email is required ; password must be 8-72 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	id, err := createUser(app.db, c.Name, c.Email, string(hash))
	if errors.Is(err, ErrUserExists) {
		writeError(w, http.StatusConflict, "Name or Email already taken", "")
		return
	}
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "internal error", "")
		return
	}

	token, err := makeToken(int(id), app.jwtSecret)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (app *App) userLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var c Credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "")
		return
	}

	c.Email = strings.TrimSpace(c.Email)
	if c.Email == "" || len(c.Password) == 0 {
		writeError(w, http.StatusBadRequest, "Validation failed",
			"Eame is required; password is required")
		return
	}

	user, err := findUserByUsername(app.db, c.Email)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(c.Password))
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			writeError(w, http.StatusUnauthorized, "Invalid credentials", "")
			return
		}
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "Internal error", "")
		return
	}

	token, err := makeToken(user.Id, app.jwtSecret)
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusInternalServerError, "internal error", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
