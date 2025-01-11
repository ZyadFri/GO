// internal/handlers/author_handler.go

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"bookstore/internal/interfaces"
	"bookstore/internal/models"
)

type AuthorHandler struct {
	authorStore interfaces.AuthorStore
}

func NewAuthorHandler(authorStore interfaces.AuthorStore) *AuthorHandler {
	return &AuthorHandler{
		authorStore: authorStore,
	}
}

func (h *AuthorHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/authors", h.handleAuthors).Methods("GET", "POST")
	router.HandleFunc("/authors/{id:[0-9]+}", h.handleAuthorByID).Methods("GET", "PUT", "DELETE")
	log.Println("Author routes registered at /api/authors")
}

func (h *AuthorHandler) handleAuthors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.listAuthors(w, r)
	case http.MethodPost:
		h.createAuthor(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AuthorHandler) handleAuthorByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAuthor(w, r, id)
	case http.MethodPut:
		h.updateAuthor(w, r, id)
	case http.MethodDelete:
		h.deleteAuthor(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AuthorHandler) createAuthor(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to create author...")

	var author models.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Decoded author data: %+v", author)

	createdAuthor, err := h.authorStore.CreateAuthor(r.Context(), author)
	if err != nil {
		log.Printf("Error creating author: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully created author: %+v", createdAuthor)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdAuthor)
}

func (h *AuthorHandler) getAuthor(w http.ResponseWriter, r *http.Request, id int) {
	author, err := h.authorStore.GetAuthor(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(author)
}

func (h *AuthorHandler) updateAuthor(w http.ResponseWriter, r *http.Request, id int) {
	var author models.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedAuthor, err := h.authorStore.UpdateAuthor(r.Context(), id, author)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedAuthor)
}

func (h *AuthorHandler) deleteAuthor(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.authorStore.DeleteAuthor(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthorHandler) listAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := h.authorStore.ListAuthors(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(authors)
}
