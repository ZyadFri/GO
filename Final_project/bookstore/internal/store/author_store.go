
package store

import (
    "context"
    "database/sql"
    "fmt"

    "bookstore/internal/interfaces"
    "bookstore/internal/models"
)

type PostgresAuthorStore struct {
    db *sql.DB
}

func NewPostgresAuthorStore(db *sql.DB) (interfaces.AuthorStore, error) {
    return &PostgresAuthorStore{db: db}, nil
}

func (s *PostgresAuthorStore) CreateAuthor(ctx context.Context, author models.Author) (models.Author, error) {
    query := `
        INSERT INTO authors (first_name, last_name, bio)
        VALUES ($1, $2, $3)
        RETURNING id
    `
    err := s.db.QueryRowContext(ctx, query,
        author.FirstName,
        author.LastName,
        author.Bio,
    ).Scan(&author.ID)
    if err != nil {
        return author, fmt.Errorf("CreateAuthor error: %w", err)
    }
    return author, nil
}

func (s *PostgresAuthorStore) GetAuthor(ctx context.Context, id int) (models.Author, error) {
    var author models.Author
    query := `
        SELECT id, first_name, last_name, bio
        FROM authors
        WHERE id = $1
    `
    err := s.db.QueryRowContext(ctx, query, id).Scan(
        &author.ID,
        &author.FirstName,
        &author.LastName,
        &author.Bio,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return author, fmt.Errorf("author not found with id: %d", id)
        }
        return author, err
    }
    return author, nil
}

func (s *PostgresAuthorStore) UpdateAuthor(ctx context.Context, id int, author models.Author) (models.Author, error) {
    query := `
        UPDATE authors
        SET first_name = $1, last_name = $2, bio = $3
        WHERE id = $4
    `
    _, err := s.db.ExecContext(ctx, query,
        author.FirstName,
        author.LastName,
        author.Bio,
        id,
    )
    if err != nil {
        return author, fmt.Errorf("UpdateAuthor error: %w", err)
    }
    author.ID = id
    return author, nil
}

func (s *PostgresAuthorStore) DeleteAuthor(ctx context.Context, id int) error {
    query := `DELETE FROM authors WHERE id = $1`
    _, err := s.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("DeleteAuthor error: %w", err)
    }
    return nil
}

func (s *PostgresAuthorStore) ListAuthors(ctx context.Context) ([]models.Author, error) {
    query := `SELECT id, first_name, last_name, bio FROM authors ORDER BY id`
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("ListAuthors error: %w", err)
    }
    defer rows.Close()

    var authors []models.Author
    for rows.Next() {
        var a models.Author
        if err := rows.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Bio); err != nil {
            return nil, err
        }
        authors = append(authors, a)
    }
    return authors, rows.Err()
}
