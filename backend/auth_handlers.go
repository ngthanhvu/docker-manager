package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"docker-ui/auth"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func AuthStatusHandler(w http.ResponseWriter, r *http.Request) {
	setupRequired, user, err := authService.Status(auth.TokenFromRequest(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"setupRequired": setupRequired,
		"authenticated": user != nil,
	}
	if user != nil {
		response["user"] = user
	}

	json.NewEncoder(w).Encode(response)
}

func AuthSetupHandler(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := authService.CreateInitialUser(request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserExists):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	json.NewEncoder(w).Encode(result)
}

func AuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := authService.Login(request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			http.Error(w, err.Error(), http.StatusUnauthorized)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	json.NewEncoder(w).Encode(result)
}

func AuthLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := authService.Logout(auth.TokenFromRequest(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
