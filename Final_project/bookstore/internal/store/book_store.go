
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"bookstore/internal/interfaces"
	"bookstore/internal/models"
)

type PostgresBookStore struct {
	db *sql.DB
}

func NewPostgresBookStore(db *sql.DB) (interfaces.BookStore, error) {
	return &PostgresBookStore{db: db}, nil
}


func (s *PostgresBookStore) CreateBook(ctx context.Context, book models.Book) (models.Book, error) {
	query := `
        INSERT INTO books (title, author_id, published_at, price, stock)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	err := s.db.QueryRowContext(ctx, query,
		book.Title,
		book.Author.ID,
		book.PublishedAt,
		book.Price,
		book.Stock,
	).Scan(&book.ID)
	if err != nil {
		return book, fmt.Errorf("CreateBook error: %w", err)
	}
	return book, nil
}


func (s *PostgresBookStore) GetBook(ctx context.Context, id int) (models.Book, error) {
	var book models.Book
	var author models.Author

	query := `
        SELECT b.id, b.title, b.published_at, b.price, b.stock,
               a.id, a.first_name, a.last_name, a.bio
        FROM books b
        JOIN authors a ON b.author_id = a.id
        WHERE b.id = $1
    `
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.Title,
		&book.PublishedAt,
		&book.Price,
		&book.Stock,
		&author.ID,
		&author.FirstName,
		&author.LastName,
		&author.Bio,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return book, fmt.Errorf("book not found with id: %d", id)
		}
		return book, err
	}
	book.Author = author
	return book, nil
}


func (s *PostgresBookStore) UpdateBook(ctx context.Context, id int, book models.Book) (models.Book, error) {
	query := `
        UPDATE books
        SET title = $1,
            author_id = $2,
            published_at = $3,
            price = $4,
            stock = $5
        WHERE id = $6
    `
	_, err := s.db.ExecContext(ctx, query,
		book.Title,
		book.Author.ID,
		book.PublishedAt,
		book.Price,
		book.Stock,
		id,
	)
	if err != nil {
		return book, fmt.Errorf("UpdateBook error: %w", err)
	}
	book.ID = id
	return book, nil
}


func (s *PostgresBookStore) DeleteBook(ctx context.Context, id int) error {
	query := `DELETE FROM books WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteBook error: %w", err)
	}
	return nil
}


func (s *PostgresBookStore) ListBooks(ctx context.Context) ([]models.Book, error) {
	query := `
        SELECT b.id, b.title, b.published_at, b.price, b.stock,
               a.id, a.first_name, a.last_name, a.bio
        FROM books b
        JOIN authors a ON b.author_id = a.id
        ORDER BY b.id
    `
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListBooks error: %w", err)
	}
	defer rows.Close()

	var result []models.Book
	for rows.Next() {
		var book models.Book
		var author models.Author
		err := rows.Scan(
			&book.ID,
			&book.Title,
			&book.PublishedAt,
			&book.Price,
			&book.Stock,
			&author.ID,
			&author.FirstName,
			&author.LastName,
			&author.Bio,
		)
		if err != nil {
			return nil, err
		}
		book.Author = author
		result = append(result, book)
	}
	return result, rows.Err()
}


func (s *PostgresBookStore) SearchBooks(ctx context.Context, criteria models.SearchCriteria) ([]models.Book, error) {
	var (
		clauses []string
		args    []interface{}
	)
	i := 1

	if criteria.Title != "" {
		clauses = append(clauses, fmt.Sprintf("LOWER(b.title) LIKE LOWER($%d)", i))
		args = append(args, "%"+criteria.Title+"%")
		i++
	}
	if criteria.Author != "" {
		clauses = append(clauses, fmt.Sprintf("(LOWER(a.first_name) LIKE LOWER($%d) OR LOWER(a.last_name) LIKE LOWER($%d))", i, i+1))
		args = append(args, "%"+criteria.Author+"%", "%"+criteria.Author+"%")
		i += 2
	}
	if criteria.MinPrice > 0 {
		clauses = append(clauses, fmt.Sprintf("b.price >= $%d", i))
		args = append(args, criteria.MinPrice)
		i++
	}
	if criteria.MaxPrice > 0 {
		clauses = append(clauses, fmt.Sprintf("b.price <= $%d", i))
		args = append(args, criteria.MaxPrice)
		i++
	}
	if criteria.PublishedBefore != nil {
		clauses = append(clauses, fmt.Sprintf("b.published_at < $%d", i))
		args = append(args, *criteria.PublishedBefore)
		i++
	}
	if criteria.PublishedAfter != nil {
		clauses = append(clauses, fmt.Sprintf("b.published_at > $%d", i))
		args = append(args, *criteria.PublishedAfter)
		i++
	}
	if criteria.MinStock > 0 {
		clauses = append(clauses, fmt.Sprintf("b.stock >= $%d", i))
		args = append(args, criteria.MinStock)
		i++
	}
	if criteria.MaxStock > 0 {
		clauses = append(clauses, fmt.Sprintf("b.stock <= $%d", i))
		args = append(args, criteria.MaxStock)
		i++
	}

	query := `
        SELECT b.id, b.title, b.published_at, b.price, b.stock,
               a.id, a.first_name, a.last_name, a.bio
        FROM books b
        JOIN authors a ON b.author_id = a.id
    `
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY b.id"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("SearchBooks error: %w", err)
	}
	defer rows.Close()

	var result []models.Book
	for rows.Next() {
		var book models.Book
		var author models.Author
		err := rows.Scan(
			&book.ID,
			&book.Title,
			&book.PublishedAt,
			&book.Price,
			&book.Stock,
			&author.ID,
			&author.FirstName,
			&author.LastName,
			&author.Bio,
		)
		if err != nil {
			return nil, err
		}
		book.Author = author
		result = append(result, book)
	}
	return result, rows.Err()
}
