

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


	bookStore interfaces.BookStore
}

func NewSalesReporter(
	orderStore interfaces.OrderStore,
	reportStore interfaces.ReportStore,
	outputDir string,
	interval time.Duration,
	bookStore interfaces.BookStore,
) (*SalesReporter, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}
	return &SalesReporter{
		orderStore:  orderStore,
		reportStore: reportStore,
		outputDir:   outputDir,
		interval:    interval,
		stopChan:    make(chan struct{}),
		bookStore:   bookStore,
	}, nil
}

func (r *SalesReporter) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()


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

func (r *SalesReporter) Stop() {
	close(r.stopChan)
}

func (r *SalesReporter) GenerateReport(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()


	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	orders, err := r.orderStore.GetOrdersInTimeRange(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to fetch orders: %v", err)
	}

	var totalRevenue float64
	bookSales := make(map[int]models.BookSales)

	for _, order := range orders {
		totalRevenue += order.TotalPrice
		for _, item := range order.Items {
			bs, ok := bookSales[item.Book.ID]
			if !ok {
				bs = models.BookSales{
					Book:     item.Book,
					Quantity: 0,
				}
			}
			bs.Quantity += item.Quantity
			bookSales[item.Book.ID] = bs
		}
	}
	var topSellingBooks []models.BookSales
	for _, bs := range bookSales {
		topSellingBooks = append(topSellingBooks, bs)
	}
	sort.Slice(topSellingBooks, func(i, j int) bool {
		return topSellingBooks[i].Quantity > topSellingBooks[j].Quantity
	})
	if len(topSellingBooks) > 10 {
		topSellingBooks = topSellingBooks[:10]
	}

	report := models.SalesReport{
		Timestamp:       endTime,
		TotalRevenue:    totalRevenue,
		TotalOrders:     len(orders),
		TopSellingBooks: topSellingBooks,
	}

	if err := r.reportStore.SaveReport(ctx, report); err != nil {
		return fmt.Errorf("failed to save report: %v", err)
	}


	filename := fmt.Sprintf("report_%s.json", endTime.Format("020060102150405"))
	filepath := filepath.Join(r.outputDir, filename)
	if err := r.writeReportToFile(filepath, report); err != nil {
		return fmt.Errorf("failed to write report to file: %v", err)
	}
	log.Printf("Generated sales report: %s", filepath)


	if len(topSellingBooks) > 0 {

		for i := 0; i < len(topSellingBooks) && i < 3; i++ {
			bookID := topSellingBooks[i].Book.ID
			err := r.adjustBookPrice(ctx, bookID)
			if err != nil {
				log.Printf("Failed to adjust price for book %d: %v", bookID, err)
			}
		}
	}

	return nil
}

func (r *SalesReporter) writeReportToFile(filepath string, report models.SalesReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}


func (r *SalesReporter) adjustBookPrice(ctx context.Context, bookID int) error {
	book, err := r.bookStore.GetBook(ctx, bookID)
	if err != nil {
		return fmt.Errorf("cannot fetch book: %w", err)
	}
	newPrice := book.Price * 1.10 
	book.Price = newPrice

	_, err = r.bookStore.UpdateBook(ctx, bookID, book)
	if err != nil {
		return fmt.Errorf("cannot update book price: %w", err)
	}
	log.Printf("Adjusted price for Book ID=%d to %.2f", bookID, newPrice)
	return nil
}
