// internal/reports/sales_reporter.go

package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
"bookstore/internal/interfaces"
	
	"bookstore/internal/models"
)

type SalesReporter struct {
	orderStore  interfaces.OrderStore
	reportStore interfaces.ReportStore
	outputDir   string
	interval    time.Duration
	mu          sync.RWMutex
	stopChan    chan struct{}
}

func NewSalesReporter(
	orderStore interfaces.OrderStore,
	reportStore interfaces.ReportStore,
	outputDir string,
	interval time.Duration,
) (*SalesReporter, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	return &SalesReporter{
		orderStore:  orderStore,
		reportStore: reportStore,
		outputDir:   outputDir,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}, nil
}

// Start begins the periodic report generation
func (r *SalesReporter) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Generate initial report
	if err := r.GenerateReport(ctx); err != nil {
		log.Printf("Error generating initial report: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stopChan:
			return nil
		case <-ticker.C:
			if err := r.GenerateReport(ctx); err != nil {
				log.Printf("Error generating report: %v", err)
			}
		}
	}
}

// Stop halts the periodic report generation
func (r *SalesReporter) Stop() {
	close(r.stopChan)
}

// GenerateReport creates a new sales report for the last 24 hours
func (r *SalesReporter) GenerateReport(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Calculate time range for the report (last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	// Get orders within the time range
	orders, err := r.orderStore.GetOrdersInTimeRange(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to fetch orders: %v", err)
	}

	// Calculate report metrics
	var totalRevenue float64
	bookSales := make(map[int]models.BookSales)

	for _, order := range orders {
		totalRevenue += order.TotalPrice

		// Aggregate book sales
		for _, item := range order.Items {
			if sales, exists := bookSales[item.Book.ID]; exists {
				sales.Quantity += item.Quantity
				bookSales[item.Book.ID] = sales
			} else {
				bookSales[item.Book.ID] = models.BookSales{
					Book:     item.Book,
					Quantity: item.Quantity,
				}
			}
		}
	}

	// Convert book sales map to sorted slice
	topSellingBooks := make([]models.BookSales, 0, len(bookSales))
	for _, sales := range bookSales {
		topSellingBooks = append(topSellingBooks, sales)
	}

	// Sort by quantity sold in descending order
	sort.Slice(topSellingBooks, func(i, j int) bool {
		return topSellingBooks[i].Quantity > topSellingBooks[j].Quantity
	})

	// Limit to top 10 books
	if len(topSellingBooks) > 10 {
		topSellingBooks = topSellingBooks[:10]
	}

	// Create the report
	report := models.SalesReport{
		Timestamp:       endTime,
		TotalRevenue:    totalRevenue,
		TotalOrders:     len(orders),
		TopSellingBooks: topSellingBooks,
	}

	// Save report to store
	if err := r.reportStore.SaveReport(ctx, report); err != nil {
		return fmt.Errorf("failed to save report: %v", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("report_%s.json", endTime.Format("020060102150405"))
	filepath := filepath.Join(r.outputDir, filename)

	// Write report to file
	if err := r.writeReportToFile(filepath, report); err != nil {
		return fmt.Errorf("failed to write report to file: %v", err)
	}

	log.Printf("Generated sales report: %s", filepath)
	return nil
}

// writeReportToFile writes the report to a JSON file
func (r *SalesReporter) writeReportToFile(filepath string, report models.SalesReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// Helper method to calculate daily statistics
func calculateDailyStats(orders []models.Order) (float64, int) {
	var totalRevenue float64
	totalOrders := len(orders)

	for _, order := range orders {
		totalRevenue += order.TotalPrice
	}

	return totalRevenue, totalOrders
}

// Helper method to get top selling books
func getTopSellingBooks(orders []models.Order, limit int) []models.BookSales {
	bookSales := make(map[int]models.BookSales)

	// Aggregate sales by book
	for _, order := range orders {
		for _, item := range order.Items {
			if sales, exists := bookSales[item.Book.ID]; exists {
				sales.Quantity += item.Quantity
				bookSales[item.Book.ID] = sales
			} else {
				bookSales[item.Book.ID] = models.BookSales{
					Book:     item.Book,
					Quantity: item.Quantity,
				}
			}
		}
	}

	// Convert to slice and sort
	sales := make([]models.BookSales, 0, len(bookSales))
	for _, s := range bookSales {
		sales = append(sales, s)
	}

	sort.Slice(sales, func(i, j int) bool {
		return sales[i].Quantity > sales[j].Quantity
	})

	// Return top N results
	if len(sales) > limit {
		sales = sales[:limit]
	}

	return sales
}
