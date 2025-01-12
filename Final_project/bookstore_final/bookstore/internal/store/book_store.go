package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"bookstore/internal/models"
)

type InMemoryBookStore struct {
	mu     sync.RWMutex
	books  map[int]models.Book
	nextID int
	dbPath string
}

func NewBookStore(dbPath string) (*InMemoryBookStore, error) {
	store := &InMemoryBookStore{
		books:  make(map[int]models.Book),
		nextID: 1,
		dbPath: dbPath,
	}
	if err := store.loadBooks(); err != nil {
		return nil, fmt.Errorf("failed to load books: %v", err)
	}
	return store, nil
}

// loadBooks: no lock needed at startup
func (s *InMemoryBookStore) loadBooks() error {
	data, err := os.ReadFile(s.dbPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var books []models.Book
	if err := json.Unmarshal(data, &books); err != nil {
		return err
	}

	for _, book := range books {
		s.books[book.ID] = book
		if book.ID >= s.nextID {
			s.nextID = book.ID + 1
		}
	}
	return nil
}

// CreateBook: lock once, write to map, then save
func (s *InMemoryBookStore) CreateBook(ctx context.Context, book models.Book) (models.Book, error) {
	select {
	case <-ctx.Done():
		return models.Book{}, ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		// 1. Check if the book already exists (matching Title + Author ID).
		for id, existingBook := range s.books {
			if existingBook.Title == book.Title && existingBook.Author.ID == book.Author.ID {
				// 2. If it exists, increment the stock.
				existingBook.Stock += book.Stock
				s.books[id] = existingBook // update in-memory map

				// 3. Save changes.
				if err := s.saveBooksUnlocked(); err != nil {
					return models.Book{}, err
				}

				// 4. Return the updated book (with incremented stock).
				return existingBook, nil
			}
		}

		// 5. If not found, create a new entry.
		book.ID = s.nextID
		s.nextID++
		s.books[book.ID] = book

		// 6. Save new book.
		if err := s.saveBooksUnlocked(); err != nil {
			// revert on error
			delete(s.books, book.ID)
			s.nextID--
			return models.Book{}, err
		}

		return book, nil
	}
}

// UpdateBook: lock, update map, then save
func (s *InMemoryBookStore) UpdateBook(ctx context.Context, id int, book models.Book) (models.Book, error) {
	select {
	case <-ctx.Done():
		return models.Book{}, ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.books[id]; !exists {
			return models.Book{}, fmt.Errorf("book not found with id: %d", id)
		}

		book.ID = id
		s.books[id] = book

		if err := s.saveBooksUnlocked(); err != nil {
			return models.Book{}, err
		}
		return book, nil
	}
}

// DeleteBook: lock, remove from map, then save
func (s *InMemoryBookStore) DeleteBook(ctx context.Context, id int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.books[id]; !exists {
			return fmt.Errorf("book not found with id: %d", id)
		}
		delete(s.books, id)

		if err := s.saveBooksUnlocked(); err != nil {
			return err
		}
		return nil
	}
}

// saveBooksUnlocked: do not lock again; assume caller holds s.mu.Lock()
func (s *InMemoryBookStore) saveBooksUnlocked() error {
	var books []models.Book
	for _, b := range s.books {
		books = append(books, b)
	}

	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal books: %v", err)
	}

	if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write books file: %v", err)
	}
	return nil
}

// GET-like methods use RLock
func (s *InMemoryBookStore) GetBook(ctx context.Context, id int) (models.Book, error) {
	select {
	case <-ctx.Done():
		return models.Book{}, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		book, exists := s.books[id]
		if !exists {
			return models.Book{}, fmt.Errorf("book not found with id: %d", id)
		}
		return book, nil
	}
}

func (s *InMemoryBookStore) ListBooks(ctx context.Context) ([]models.Book, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		books := make([]models.Book, 0, len(s.books))
		for _, b := range s.books {
			books = append(books, b)
		}
		return books, nil
	}
}

func (s *InMemoryBookStore) SearchBooks(ctx context.Context, criteria models.SearchCriteria) ([]models.Book, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		var results []models.Book
		for _, b := range s.books {
			if matchesCriteria(b, criteria) {
				results = append(results, b)
			}
		}
		return results, nil
	}
}

// The old saveBooks() method is replaced by saveBooksUnlocked() to avoid re-locking
// The matchesCriteria function is unchanged
func matchesCriteria(book models.Book, c models.SearchCriteria) bool {
	if c.Title != "" && !strings.Contains(strings.ToLower(book.Title), strings.ToLower(c.Title)) {
		return false
	}
	if c.Author != "" {
		fullName := book.Author.FirstName + " " + book.Author.LastName
		if !strings.Contains(strings.ToLower(fullName), strings.ToLower(c.Author)) {
			return false
		}
	}
	if len(c.Genres) > 0 {
		matched := false
		for _, wantGenre := range c.Genres {
			for _, haveGenre := range book.Genres {
				if strings.EqualFold(wantGenre, haveGenre) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}
	if c.MinPrice > 0 && book.Price < c.MinPrice {
		return false
	}
	if c.MaxPrice > 0 && book.Price > c.MaxPrice {
		return false
	}
	return true
}
