package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"bookstore/internal/interfaces"
	"bookstore/internal/models"
)

type InMemoryReportStore struct {
	mu      sync.RWMutex
	reports []models.SalesReport
	dbPath  string
}

func NewReportStore(dbPath string) (*InMemoryReportStore, error) {
	store := &InMemoryReportStore{
		reports: make([]models.SalesReport, 0),
		dbPath:  dbPath,
	}
	if err := store.loadReports(); err != nil {
		return nil, fmt.Errorf("failed to load reports: %v", err)
	}
	return store, nil
}

// loadReports: no lock needed at startup
func (s *InMemoryReportStore) loadReports() error {
	data, err := os.ReadFile(s.dbPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var reports []models.SalesReport
	if err := json.Unmarshal(data, &reports); err != nil {
		return err
	}
	s.reports = reports
	return nil
}

// SaveReport: lock, append to s.reports, then save
func (s *InMemoryReportStore) SaveReport(ctx context.Context, report models.SalesReport) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.reports = append(s.reports, report)

		if err := s.saveReportsUnlocked(); err != nil {
			return err
		}
		return nil
	}
}

// saveReportsUnlocked: no second lock
func (s *InMemoryReportStore) saveReportsUnlocked() error {
	data, err := json.MarshalIndent(s.reports, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal reports: %v", err)
	}
	if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write reports file: %v", err)
	}
	return nil
}

// GetReports: read lock
func (s *InMemoryReportStore) GetReports(ctx context.Context, start, end time.Time) ([]models.SalesReport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		var out []models.SalesReport
		for _, r := range s.reports {
			if (r.Timestamp.Equal(start) || r.Timestamp.After(start)) &&
				(r.Timestamp.Equal(end) || r.Timestamp.Before(end)) {
				out = append(out, r)
			}
		}
		return out, nil
	}
}

// Ensure it implements interfaces.ReportStore
var _ interfaces.ReportStore = (*InMemoryReportStore)(nil)
