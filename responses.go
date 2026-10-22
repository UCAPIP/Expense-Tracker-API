package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

var (
	ErrNotFound   = errors.New("Expenses not found")
	ErrUserExists = errors.New("Name or email already taken")
	ErrForbidden  = errors.New("Forbidden")
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println(err)
	}
}

func writeError(w http.ResponseWriter, status int, msg, details string) {
	writeJSON(w, status, ErrorResponse{Error: msg, Details: details})
}

func intParam(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, errors.New("Must be a positive integer")
	}
	return n, nil
}
