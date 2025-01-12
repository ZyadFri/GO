package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"bookstore/internal/models"

)

type InMemoryAuthorStore struct {
	mu      sync.RWMutex
	authors map[int]models.Author
	nextID  int
	dbPath  string
}

// Constructor
func NewAuthorStore(dbPath string) (*InMemoryAuthorStore, error) {
	store := &InMemoryAuthorStore{
		authors: make(map[int]models.Author),
		nextID:  1,
		dbPath:  dbPath,
	}

	if err := store.loadAuthors(); err != nil {
		return nil, fmt.Errorf("failed to load authors: %v", err)
	}
	return store, nil
}

// ========== LOAD AUTHORS ==========
func (s *InMemoryAuthorStore) loadAuthors() error {
	data, err := os.ReadFile(s.dbPath)
	if os.IsNotExist(err) {
		return nil // file doesn't exist => no authors yet
	}
	if err != nil {
		return err
	}

	var authors []models.Author
	if err := json.Unmarshal(data, &authors); err != nil {
		return err
	}

	for _, author := range authors {
		s.authors[author.ID] = author
		if author.ID >= s.nextID {
			s.nextID = author.ID + 1
		}
	}
	return nil
}

// ========== CREATE AUTHOR ==========
func (s *InMemoryAuthorStore) CreateAuthor(ctx context.Context, author models.Author) (models.Author, error) {
	select {
	case <-ctx.Done():
		return models.Author{}, ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if author.FirstName == "" || author.LastName == "" {
			return models.Author{}, fmt.Errorf("first name and last name are required")
		}

		author.ID = s.nextID
		s.nextID++
		s.authors[author.ID] = author

		if err := s.saveAuthorsUnlocked(); err != nil {
			// revert
			delete(s.authors, author.ID)
			s.nextID--
			return models.Author{}, err
		}
		return author, nil
	}
}

// ========== SAVE AUTHORS (unlocked) ==========
func (s *InMemoryAuthorStore) saveAuthorsUnlocked() error {
	authorsSlice := make([]models.Author, 0, len(s.authors))
	for _, a := range s.authors {
		authorsSlice = append(authorsSlice, a)
	}

	data, err := json.MarshalIndent(authorsSlice, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal authors: %v", err)
	}

	if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write authors file: %v", err)
	}
	return nil
}

// ========== GET AUTHOR ==========
func (s *InMemoryAuthorStore) GetAuthor(ctx context.Context, id int) (models.Author, error) {
	select {
	case <-ctx.Done():
		return models.Author{}, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		author, exists := s.authors[id]
		if !exists {
			return models.Author{}, fmt.Errorf("author not found with id: %d", id)
		}
		return author, nil
	}
}

// ========== UPDATE AUTHOR ==========
func (s *InMemoryAuthorStore) UpdateAuthor(ctx context.Context, id int, author models.Author) (models.Author, error) {
	select {
	case <-ctx.Done():
		return models.Author{}, ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.authors[id]; !exists {
			return models.Author{}, fmt.Errorf("author not found with id: %d", id)
		}

		author.ID = id
		s.authors[id] = author

		if err := s.saveAuthorsUnlocked(); err != nil {
			return models.Author{}, err
		}
		return author, nil
	}
}

// ========== DELETE AUTHOR ==========
func (s *InMemoryAuthorStore) DeleteAuthor(ctx context.Context, id int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.authors[id]; !exists {
			return fmt.Errorf("author not found with id: %d", id)
		}

		delete(s.authors, id)
		if err := s.saveAuthorsUnlocked(); err != nil {
			return err
		}
		return nil
	}
}

// ========== LIST AUTHORS ==========
func (s *InMemoryAuthorStore) ListAuthors(ctx context.Context) ([]models.Author, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		authorsSlice := make([]models.Author, 0, len(s.authors))
		for _, a := range s.authors {
			authorsSlice = append(authorsSlice, a)
		}
		return authorsSlice, nil
	}
}
