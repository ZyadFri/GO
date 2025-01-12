
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bookstore/internal/interfaces"
	"bookstore/internal/models"
)

type PostgresCustomerStore struct {
	db *sql.DB
}

func NewPostgresCustomerStore(db *sql.DB) (interfaces.CustomerStore, error) {
	return &PostgresCustomerStore{db: db}, nil
}

func (s *PostgresCustomerStore) CreateCustomer(ctx context.Context, customer models.Customer) (models.Customer, error) {
	query := `
        INSERT INTO customers (name, email, street, city, state, postal_code, country, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `
	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		customer.Name,
		customer.Email,
		customer.Address.Street,
		customer.Address.City,
		customer.Address.State,
		customer.Address.PostalCode,
		customer.Address.Country,
		now,
	).Scan(&customer.ID)
	if err != nil {
		return customer, fmt.Errorf("CreateCustomer error: %w", err)
	}
	customer.CreatedAt = now
	return customer, nil
}

func (s *PostgresCustomerStore) GetCustomer(ctx context.Context, id int) (models.Customer, error) {
	var c models.Customer
	query := `
        SELECT id, name, email, street, city, state, postal_code, country, created_at
        FROM customers
        WHERE id = $1
    `
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.Address.Street,
		&c.Address.City,
		&c.Address.State,
		&c.Address.PostalCode,
		&c.Address.Country,
		&c.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c, fmt.Errorf("customer not found with id: %d", id)
		}
		return c, err
	}
	return c, nil
}

func (s *PostgresCustomerStore) UpdateCustomer(ctx context.Context, id int, customer models.Customer) (models.Customer, error) {
	
	existing, err := s.GetCustomer(ctx, id)
	if err != nil {
		return customer, err
	}

	query := `
        UPDATE customers
        SET name = $1, email = $2, street = $3, city = $4,
            state = $5, postal_code = $6, country = $7
        WHERE id = $8
    `
	_, err = s.db.ExecContext(ctx, query,
		customer.Name,
		customer.Email,
		customer.Address.Street,
		customer.Address.City,
		customer.Address.State,
		customer.Address.PostalCode,
		customer.Address.Country,
		id,
	)
	if err != nil {
		return customer, fmt.Errorf("UpdateCustomer error: %w", err)
	}
	customer.ID = id
	customer.CreatedAt = existing.CreatedAt
	return customer, nil
}

func (s *PostgresCustomerStore) DeleteCustomer(ctx context.Context, id int) error {
	query := `DELETE FROM customers WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteCustomer error: %w", err)
	}
	return nil
}

func (s *PostgresCustomerStore) ListCustomers(ctx context.Context) ([]models.Customer, error) {
	query := `
        SELECT id, name, email, street, city, state, postal_code, country, created_at
        FROM customers
        ORDER BY id
    `
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListCustomers error: %w", err)
	}
	defer rows.Close()

	var results []models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Email,
			&c.Address.Street,
			&c.Address.City,
			&c.Address.State,
			&c.Address.PostalCode,
			&c.Address.Country,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, rows.Err()
}
