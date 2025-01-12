

package handlers

import (
    "encoding/json"
    "net/http"

    "bookstore/internal/auth"
    "github.com/gorilla/mux"
)

type AuthHandler struct {
    JWTManager *auth.JWTManager
}


func NewAuthHandler(jwtManager *auth.JWTManager) *AuthHandler {
    return &AuthHandler{
        JWTManager: jwtManager,
    }
}

func (h *AuthHandler) RegisterRoutes(router *mux.Router) {

    router.HandleFunc("/login", h.handleLogin).Methods("POST")
}

type loginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
    var req loginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if req.Username == "admin" && req.Password == "password" {
        token, err := h.JWTManager.Generate(req.Username)
        if err != nil {
            http.Error(w, "failed to generate token", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"token":"` + token + `"}`))
    } else {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
    }
}
