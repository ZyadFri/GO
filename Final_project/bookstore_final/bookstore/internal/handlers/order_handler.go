// internal/handlers/order_handler.go

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"bookstore/internal/models"
	
"bookstore/internal/interfaces"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	orderStore interfaces.OrderStore
	bookStore  interfaces.BookStore // Used to verify book availability
}

func NewOrderHandler(orderStore interfaces.OrderStore, bookStore interfaces.BookStore) *OrderHandler {
	return &OrderHandler{
		orderStore: orderStore,
		bookStore:  bookStore,
	}
}

func (h *OrderHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/orders", h.handleOrders).Methods("GET", "POST")
	router.HandleFunc("/orders/{id:[0-9]+}", h.handleOrderByID).Methods("GET", "PUT", "DELETE")
}

func (h *OrderHandler) handleOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.listOrders(w, r)
	case http.MethodPost:
		h.createOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *OrderHandler) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getOrder(w, r, id)
	case http.MethodPut:
		h.updateOrder(w, r, id)
	case http.MethodDelete:
		h.deleteOrder(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify book availability and update stock
	for _, item := range order.Items {
		book, err := h.bookStore.GetBook(r.Context(), item.Book.ID)
		if err != nil {
			http.Error(w, "Book not found: "+err.Error(), http.StatusBadRequest)
			return
		}

		if book.Stock < item.Quantity {
			http.Error(w, "Insufficient stock for book: "+book.Title, http.StatusBadRequest)
			return
		}

		// Update book stock
		book.Stock -= item.Quantity
		_, err = h.bookStore.UpdateBook(r.Context(), book.ID, book)
		if err != nil {
			http.Error(w, "Failed to update book stock: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	createdOrder, err := h.orderStore.CreateOrder(r.Context(), order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdOrder)
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request, id int) {
	order, err := h.orderStore.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) updateOrder(w http.ResponseWriter, r *http.Request, id int) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the existing order to handle stock changes
	existingOrder, err := h.orderStore.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create a map of existing order items for easy lookup
	existingItems := make(map[int]models.OrderItem)
	for _, item := range existingOrder.Items {
		existingItems[item.Book.ID] = item
	}

	// Verify and update book stock for new/modified items
	for _, item := range order.Items {
		book, err := h.bookStore.GetBook(r.Context(), item.Book.ID)
		if err != nil {
			http.Error(w, "Book not found: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Calculate the stock change
		existingQty := 0
		if existingItem, ok := existingItems[item.Book.ID]; ok {
			existingQty = existingItem.Quantity
		}
		stockChange := item.Quantity - existingQty

		if book.Stock < stockChange {
			http.Error(w, "Insufficient stock for book: "+book.Title, http.StatusBadRequest)
			return
		}

		// Update book stock
		book.Stock -= stockChange
		_, err = h.bookStore.UpdateBook(r.Context(), book.ID, book)
		if err != nil {
			http.Error(w, "Failed to update book stock: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	updatedOrder, err := h.orderStore.UpdateOrder(r.Context(), id, order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedOrder)
}

func (h *OrderHandler) deleteOrder(w http.ResponseWriter, r *http.Request, id int) {
	// Get the order to restore book stock
	order, err := h.orderStore.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Restore book stock
	for _, item := range order.Items {
		book, err := h.bookStore.GetBook(r.Context(), item.Book.ID)
		if err != nil {
			http.Error(w, "Book not found: "+err.Error(), http.StatusInternalServerError)
			return
		}

		book.Stock += item.Quantity
		_, err = h.bookStore.UpdateBook(r.Context(), book.ID, book)
		if err != nil {
			http.Error(w, "Failed to restore book stock: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := h.orderStore.DeleteOrder(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orderStore.ListOrders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(orders)
}
