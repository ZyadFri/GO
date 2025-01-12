// internal/handlers/customer_handler.go

package handlers

import (
	"encoding/json"
	"bookstore/internal/models"
	
	"net/http"
	"strconv"
	"strings"

    "bookstore/internal/interfaces" 
	"github.com/gorilla/mux"
)

type CustomerHandler struct {
	customerStore interfaces.CustomerStore
}

func NewCustomerHandler(customerStore interfaces.CustomerStore) *CustomerHandler {
	return &CustomerHandler{
		customerStore: customerStore,
	}
}
func (h *CustomerHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/customers", h.handleCustomers).Methods("GET", "POST")
	router.HandleFunc("/customers/{id:[0-9]+}", h.handleCustomerByID).Methods("GET", "PUT", "DELETE")
}

func (h *CustomerHandler) handleCustomers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.listCustomers(w, r)
	case http.MethodPost:
		h.createCustomer(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CustomerHandler) handleCustomerByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getCustomer(w, r, id)
	case http.MethodPut:
		h.updateCustomer(w, r, id)
	case http.MethodDelete:
		h.deleteCustomer(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CustomerHandler) createCustomer(w http.ResponseWriter, r *http.Request) {
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdCustomer, err := h.customerStore.CreateCustomer(r.Context(), customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdCustomer)
}

func (h *CustomerHandler) getCustomer(w http.ResponseWriter, r *http.Request, id int) {
	customer, err := h.customerStore.GetCustomer(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(customer)
}

func (h *CustomerHandler) updateCustomer(w http.ResponseWriter, r *http.Request, id int) {
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedCustomer, err := h.customerStore.UpdateCustomer(r.Context(), id, customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedCustomer)
}

func (h *CustomerHandler) deleteCustomer(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.customerStore.DeleteCustomer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomerHandler) listCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := h.customerStore.ListCustomers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(customers)
}
