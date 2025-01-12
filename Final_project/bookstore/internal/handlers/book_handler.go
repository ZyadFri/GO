

package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/gorilla/mux"

    "bookstore/internal/interfaces"
    "bookstore/internal/models"
)

type BookHandler struct {
    bookStore interfaces.BookStore
}

func NewBookHandler(bookStore interfaces.BookStore) *BookHandler {
    return &BookHandler{bookStore: bookStore}
}


func (h *BookHandler) RegisterRoutes(router *mux.Router, mw func(http.Handler) http.Handler) {

    router.Handle("/books", mw(http.HandlerFunc(h.handleBooks))).
        Methods("GET", "POST")

    router.Handle("/books/{id:[0-9]+}", mw(http.HandlerFunc(h.handleBookByID))).
        Methods("GET", "PUT", "DELETE")
}

func (h *BookHandler) handleBooks(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    switch r.Method {
    case http.MethodGet:
        h.searchBooks(w, r)
    case http.MethodPost:
        h.createBook(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (h *BookHandler) handleBookByID(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "Invalid book ID", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodGet:
        h.getBook(w, r, id)
    case http.MethodPut:
        h.updateBook(w, r, id)
    case http.MethodDelete:
        h.deleteBook(w, r, id)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (h *BookHandler) createBook(w http.ResponseWriter, r *http.Request) {
    var book models.Book
    if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    createdBook, err := h.bookStore.CreateBook(r.Context(), book)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(createdBook)
}

func (h *BookHandler) getBook(w http.ResponseWriter, r *http.Request, id int) {
    book, err := h.bookStore.GetBook(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    json.NewEncoder(w).Encode(book)
}

func (h *BookHandler) updateBook(w http.ResponseWriter, r *http.Request, id int) {
    var book models.Book
    if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    updatedBook, err := h.bookStore.UpdateBook(r.Context(), id, book)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(updatedBook)
}

func (h *BookHandler) deleteBook(w http.ResponseWriter, r *http.Request, id int) {
    if err := h.bookStore.DeleteBook(r.Context(), id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}


func (h *BookHandler) searchBooks(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    criteria := models.SearchCriteria{
        Title:    query.Get("title"),
        Author:   query.Get("author"),
        Genres:   query["genres"], 
        MinPrice: parseFloat(query.Get("min_price"), 0),
        MaxPrice: parseFloat(query.Get("max_price"), 0),
    }


    publishedBefore := query.Get("published_before")
    publishedAfter  := query.Get("published_after")
    minStock        := query.Get("min_stock")
    maxStock        := query.Get("max_stock")

    if publishedBefore != "" {
        t, err := time.Parse(time.RFC3339, publishedBefore)
        if err == nil {
            criteria.PublishedBefore = &t
        }
    }
    if publishedAfter != "" {
        t, err := time.Parse(time.RFC3339, publishedAfter)
        if err == nil {
            criteria.PublishedAfter = &t
        }
    }
    if minStock != "" {
        criteria.MinStock = parseInt(minStock, 0)
    }
    if maxStock != "" {
        criteria.MaxStock = parseInt(maxStock, 0)
    }

    books, err := h.bookStore.SearchBooks(r.Context(), criteria)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(books)
}


func parseFloat(s string, defaultVal float64) float64 {
    if s == "" {
        return defaultVal
    }
    val, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return defaultVal
    }
    return val
}

func parseInt(s string, defaultVal int) int {
    if s == "" {
        return defaultVal
    }
    val, err := strconv.Atoi(s)
    if err != nil {
        return defaultVal
    }
    return val
}
