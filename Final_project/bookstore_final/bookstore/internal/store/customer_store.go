package store

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"

    "bookstore/internal/models"
)

type InMemoryCustomerStore struct {
    mu        sync.RWMutex
    customers map[int]models.Customer
    nextID    int
    dbPath    string
}

func NewCustomerStore(dbPath string) (*InMemoryCustomerStore, error) {
    store := &InMemoryCustomerStore{
        customers: make(map[int]models.Customer),
        nextID:    1,
        dbPath:    dbPath,
    }
    if err := store.loadCustomers(); err != nil {
        return nil, fmt.Errorf("failed to load customers: %v", err)
    }
    return store, nil
}

// loadCustomers: no lock needed at startup
func (s *InMemoryCustomerStore) loadCustomers() error {
    data, err := os.ReadFile(s.dbPath)
    if os.IsNotExist(err) {
        return nil
    }
    if err != nil {
        return err
    }

    var customers []models.Customer
    if err := json.Unmarshal(data, &customers); err != nil {
        return err
    }

    for _, c := range customers {
        s.customers[c.ID] = c
        if c.ID >= s.nextID {
            s.nextID = c.ID + 1
        }
    }
    return nil
}

// CreateCustomer: lock, add to map, then save
func (s *InMemoryCustomerStore) CreateCustomer(ctx context.Context, customer models.Customer) (models.Customer, error) {
    select {
    case <-ctx.Done():
        return models.Customer{}, ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        customer.ID = s.nextID
        customer.CreatedAt = time.Now()
        s.nextID++
        s.customers[customer.ID] = customer

        if err := s.saveCustomersUnlocked(); err != nil {
            // revert
            delete(s.customers, customer.ID)
            s.nextID--
            return models.Customer{}, err
        }
        return customer, nil
    }
}

// UpdateCustomer: lock, update map, then save
func (s *InMemoryCustomerStore) UpdateCustomer(ctx context.Context, id int, customer models.Customer) (models.Customer, error) {
    select {
    case <-ctx.Done():
        return models.Customer{}, ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        existing, exists := s.customers[id]
        if !exists {
            return models.Customer{}, fmt.Errorf("customer not found with id: %d", id)
        }

        // Keep the same CreatedAt
        customer.ID = id
        customer.CreatedAt = existing.CreatedAt
        s.customers[id] = customer

        if err := s.saveCustomersUnlocked(); err != nil {
            return models.Customer{}, err
        }
        return customer, nil
    }
}

// DeleteCustomer: lock, remove from map, then save
func (s *InMemoryCustomerStore) DeleteCustomer(ctx context.Context, id int) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        if _, exists := s.customers[id]; !exists {
            return fmt.Errorf("customer not found with id: %d", id)
        }
        delete(s.customers, id)

        if err := s.saveCustomersUnlocked(); err != nil {
            return err
        }
        return nil
    }
}

// saveCustomersUnlocked: no additional lock
func (s *InMemoryCustomerStore) saveCustomersUnlocked() error {
    var customers []models.Customer
    for _, c := range s.customers {
        customers = append(customers, c)
    }

    data, err := json.MarshalIndent(customers, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal customers: %v", err)
    }

    if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write customers file: %v", err)
    }
    return nil
}

// GET-like methods use RLock
func (s *InMemoryCustomerStore) GetCustomer(ctx context.Context, id int) (models.Customer, error) {
    select {
    case <-ctx.Done():
        return models.Customer{}, ctx.Err()
    default:
        s.mu.RLock()
        defer s.mu.RUnlock()

        c, exists := s.customers[id]
        if !exists {
            return models.Customer{}, fmt.Errorf("customer not found with id: %d", id)
        }
        return c, nil
    }
}

func (s *InMemoryCustomerStore) ListCustomers(ctx context.Context) ([]models.Customer, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        s.mu.RLock()
        defer s.mu.RUnlock()

        var list []models.Customer
        for _, c := range s.customers {
            list = append(list, c)
        }
        return list, nil
    }
}
