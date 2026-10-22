package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (app *App) getExpenses(w http.ResponseWriter, r *http.Request, userId int) {

	tasks, err := loading(app.db, userId)
	if err != nil {
		log.Printf("ERROR: Unable to load tasks from DB: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "Unable to load data.")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

type ExpensesInput struct {
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Category    string `json:"category"`
}

func validate(input *ExpensesInput) []string {
	var problems []string
	if strings.TrimSpace(input.Description) == "" {
		problems = append(problems, "Description is required")
	}
	if strings.TrimSpace(input.Amount) == "" {
		problems = append(problems, "Amount is required")
	}
	return problems
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

//func (app *App) updateExpensesById(w http.ResponseWriter, r *http.Request, userId int) {
//	taskIdStr := r.PathValue("id")
//	taskId, err := strconv.Atoi(taskIdStr)
//	if err != nil {
//		writeError(w, http.StatusBadRequest, "Bad Request", "The 'id' parameter must be a valid integer number.")
//		return
//	}
//
//	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
//	var input ExpensesInput
//	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
//		writeError(w, http.StatusBadRequest, "Invalid JSON body", "")
//		return
//	}
//
//	if problems := validate(&input); len(problems) > 0 {
//		writeError(w, http.StatusBadRequest, "Validation failed", strings.Join(problems, "; "))
//		return
//	}
//
//	owner, err := taskOwner(app.db, taskId)
//	if errors.Is(err, ErrNotFound) {
//		writeError(w, http.StatusNotFound, "Task not found", "")
//		return
//	}
//	if err != nil {
//		log.Println(err)
//		writeError(w, http.StatusInternalServerError, "Internal error", "")
//		return
//	}
//	if owner != userId {
//		writeError(w, http.StatusForbidden, "Forbidden", "")
//		return
//	}
//
//	err = editing(app.db, &input, taskId, userId)
//	if errors.Is(err, ErrNotFound) {
//		writeError(w, http.StatusNotFound, "Task not found", "")
//		return
//	}
//	if err != nil {
//		log.Println(err)
//		writeError(w, http.StatusInternalServerError, "Internal error", "")
//		return
//	}
//
//	updatedTask, err := finding(app.db, taskId, userId)
//	if err != nil {
//		log.Println(err)
//		writeError(w, http.StatusInternalServerError, "Internal error", "")
//		return
//	}
//
//	writeJSON(w, http.StatusOK, updatedTask)
//
//}
//
//func (app *App) deleteExpensesById(w http.ResponseWriter, r *http.Request, userId int) {
//	taskIdStr := r.PathValue("id")
//	taskId, err := strconv.Atoi(taskIdStr)
//	if err != nil {
//		writeError(w, http.StatusBadRequest, "Bad Request", "The 'id' parameter must be a valid integer number.")
//		return
//	}
//
//	owner, err := taskOwner(app.db, taskId)
//	if errors.Is(err, ErrNotFound) {
//		writeError(w, http.StatusNotFound, "Task not found", "")
//		return
//	}
//	if err != nil {
//		log.Println(err)
//		writeError(w, http.StatusInternalServerError, "Internal error", "")
//		return
//	}
//	if owner != userId {
//		writeError(w, http.StatusForbidden, "Forbidden", "")
//		return
//	}
//
//	err = deleting(app.db, taskId, userId)
//	if errors.Is(err, ErrNotFound) {
//		writeError(w, http.StatusNotFound, "Task not found", "")
//		return
//	}
//	if err != nil {
//		log.Println(err)
//		writeError(w, http.StatusInternalServerError, "Internal error", "")
//		return
//	}
//
//	w.WriteHeader(http.StatusNoContent)
//}

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
