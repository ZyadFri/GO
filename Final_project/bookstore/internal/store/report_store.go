
package store

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "bookstore/internal/interfaces"
    "bookstore/internal/models"
)

type PostgresReportStore struct {
    db *sql.DB
}

func NewPostgresReportStore(db *sql.DB) (interfaces.ReportStore, error) {
    return &PostgresReportStore{db: db}, nil
}

func (s *PostgresReportStore) SaveReport(ctx context.Context, report models.SalesReport) error {
 
    fmt.Println("SaveReport called, but not storing in DB (no table).")
    return nil
}

func (s *PostgresReportStore) GetReports(ctx context.Context, start, end time.Time) ([]models.SalesReport, error) {

    return []models.SalesReport{}, nil
}
