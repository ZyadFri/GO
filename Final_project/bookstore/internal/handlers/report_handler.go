// internal/handlers/report_handler.go

package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"bookstore/internal/models"
	"bookstore/internal/interfaces"

	"github.com/gorilla/mux"
)

type ReportHandler struct {
	reportStore interfaces.ReportStore
}

func NewReportHandler(reportStore interfaces.ReportStore) *ReportHandler {
	return &ReportHandler{
		reportStore: reportStore,
	}
}

func (h *ReportHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/reports/sales", h.handleSalesReports).Methods("GET")
}

func (h *ReportHandler) handleSalesReports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse date range from query parameters
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, "Invalid date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get reports within the date range
	reports, err := h.reportStore.GetReports(r.Context(), startDate, endDate)
	if err != nil {
		http.Error(w, "Failed to fetch reports: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(struct {
		Reports []models.SalesReport `json:"reports"`
	}{
		Reports: reports,
	})
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	query := r.URL.Query()

	startStr := query.Get("start_date")
	if startStr == "" {
		startStr = time.Now().AddDate(0, -1, 0).Format("2006-01-02") // Default to last month
	}

	endStr := query.Get("end_date")
	if endStr == "" {
		endStr = time.Now().Format("2006-01-02") // Default to today
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// Ensure start date is before end date
	if start.After(end) {
		return time.Time{}, time.Time{}, err
	}

	// Set end date to end of day
	end = end.Add(24*time.Hour - time.Second)

	return start, end, nil
}
