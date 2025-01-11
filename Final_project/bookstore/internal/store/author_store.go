package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// Load authors from file (no locking needed on startup)
func (s *InMemoryAuthorStore) loadAuthors() error {
	// If file doesn't exist, start empty
	data, err := os.ReadFile(s.dbPath)
	if os.IsNotExist(err) {
		return nil
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

// Create a new author (write lock)
func (s *InMemoryAuthorStore) CreateAuthor(ctx context.Context, author models.Author) (models.Author, error) {
	log.Println("CreateAuthor: start")

	s.mu.Lock()
	log.Println("CreateAuthor: lock acquired")
	defer s.mu.Unlock()

	// Validate
	if author.FirstName == "" || author.LastName == "" {
		log.Println("CreateAuthor: missing name fields")
		return models.Author{}, fmt.Errorf("first name and last name are required")
	}

	// Assign new ID
	author.ID = s.nextID
	s.nextID++
	s.authors[author.ID] = author
	log.Printf("CreateAuthor: assigned ID = %d, calling saveAuthors\n", author.ID)

	// Save to file (no additional locking here)
	if err := s.saveAuthorsUnlocked(); err != nil {
		log.Println("CreateAuthor: saveAuthors error:", err)
		// revert
		delete(s.authors, author.ID)
		s.nextID--
		return models.Author{}, err
	}

	log.Println("CreateAuthor: saveAuthors returned successfully, done.")
	return author, nil
}

// saveAuthorsUnlocked: Called only while we already hold s.mu.Lock()
func (s *InMemoryAuthorStore) saveAuthorsUnlocked() error {
	log.Println("saveAuthors: start (unlocked)")

	// We assume we already hold a write lock at this point.
	authors := make([]models.Author, 0, len(s.authors))
	for _, author := range s.authors {
		authors = append(authors, author)
	}
	log.Printf("saveAuthors: about to marshal %d authors\n", len(authors))

	data, err := json.MarshalIndent(authors, "", "  ")
	if err != nil {
		log.Println("saveAuthors: marshal error:", err)
		return fmt.Errorf("failed to marshal authors: %v", err)
	}

	log.Println("saveAuthors: about to write file", s.dbPath)
	err = os.WriteFile(s.dbPath, data, 0644)
	if err != nil {
		log.Println("saveAuthors: write file error:", err)
		return fmt.Errorf("failed to write authors file: %v", err)
	}

	log.Println("saveAuthors: file write done. Returning success.")
	return nil
}

// Get a single author (read lock)
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

// Update an author (write lock)
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

// Delete an author (write lock)
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

// List all authors (read lock)
func (s *InMemoryAuthorStore) ListAuthors(ctx context.Context) ([]models.Author, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		authors := make([]models.Author, 0, len(s.authors))
		for _, a := range s.authors {
			authors = append(authors, a)
		}
		return authors, nil
	}
}
