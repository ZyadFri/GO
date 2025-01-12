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

type InMemoryOrderStore struct {
    mu     sync.RWMutex
    orders map[int]models.Order
    nextID int
    dbPath string
}

func NewOrderStore(dbPath string) (*InMemoryOrderStore, error) {
    store := &InMemoryOrderStore{
        orders: make(map[int]models.Order),
        nextID: 1,
        dbPath: dbPath,
    }
    if err := store.loadOrders(); err != nil {
        return nil, fmt.Errorf("failed to load orders: %v", err)
    }
    return store, nil
}

// loadOrders: no lock needed on startup
func (s *InMemoryOrderStore) loadOrders() error {
    data, err := os.ReadFile(s.dbPath)
    if os.IsNotExist(err) {
        return nil
    }
    if err != nil {
        return err
    }

    var orders []models.Order
    if err := json.Unmarshal(data, &orders); err != nil {
        return err
    }

    for _, o := range orders {
        s.orders[o.ID] = o
        if o.ID >= s.nextID {
            s.nextID = o.ID + 1
        }
    }
    return nil
}

// CreateOrder: lock, add to map, then save
func (s *InMemoryOrderStore) CreateOrder(ctx context.Context, order models.Order) (models.Order, error) {
    select {
    case <-ctx.Done():
        return models.Order{}, ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        // Calculate total price
        var total float64
        for _, item := range order.Items {
            total += item.Book.Price * float64(item.Quantity)
        }
        order.TotalPrice = total
        order.ID = s.nextID
        order.CreatedAt = time.Now()
        order.Status = "pending"
        s.nextID++

        s.orders[order.ID] = order
        if err := s.saveOrdersUnlocked(); err != nil {
            // revert
            delete(s.orders, order.ID)
            s.nextID--
            return models.Order{}, err
        }
        return order, nil
    }
}

// UpdateOrder: lock, update map, then save
func (s *InMemoryOrderStore) UpdateOrder(ctx context.Context, id int, order models.Order) (models.Order, error) {
    select {
    case <-ctx.Done():
        return models.Order{}, ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        existing, exists := s.orders[id]
        if !exists {
            return models.Order{}, fmt.Errorf("order not found with id: %d", id)
        }

        // preserve ID, CreatedAt
        order.ID = id
        order.CreatedAt = existing.CreatedAt

        // recalc total
        var total float64
        for _, item := range order.Items {
            total += item.Book.Price * float64(item.Quantity)
        }
        order.TotalPrice = total

        s.orders[id] = order
        if err := s.saveOrdersUnlocked(); err != nil {
            return models.Order{}, err
        }
        return order, nil
    }
}

// DeleteOrder: lock, remove from map, then save
func (s *InMemoryOrderStore) DeleteOrder(ctx context.Context, id int) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        s.mu.Lock()
        defer s.mu.Unlock()

        if _, exists := s.orders[id]; !exists {
            return fmt.Errorf("order not found with id: %d", id)
        }
        delete(s.orders, id)

        if err := s.saveOrdersUnlocked(); err != nil {
            return err
        }
        return nil
    }
}

// saveOrdersUnlocked: no second lock
func (s *InMemoryOrderStore) saveOrdersUnlocked() error {
    var orders []models.Order
    for _, o := range s.orders {
        orders = append(orders, o)
    }

    data, err := json.MarshalIndent(orders, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal orders: %v", err)
    }

    if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write orders file: %v", err)
    }
    return nil
}

// GET-like methods with RLock
func (s *InMemoryOrderStore) GetOrder(ctx context.Context, id int) (models.Order, error) {
    select {
    case <-ctx.Done():
        return models.Order{}, ctx.Err()
    default:
        s.mu.RLock()
        defer s.mu.RUnlock()

        o, exists := s.orders[id]
        if !exists {
            return models.Order{}, fmt.Errorf("order not found with id: %d", id)
        }
        return o, nil
    }
}

func (s *InMemoryOrderStore) ListOrders(ctx context.Context) ([]models.Order, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        s.mu.RLock()
        defer s.mu.RUnlock()

        var list []models.Order
        for _, o := range s.orders {
            list = append(list, o)
        }
        return list, nil
    }
}

// Additional method to filter by time range
func (s *InMemoryOrderStore) GetOrdersInTimeRange(ctx context.Context, start, end time.Time) ([]models.Order, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        s.mu.RLock()
        defer s.mu.RUnlock()

        var orders []models.Order
        for _, o := range s.orders {
            if (o.CreatedAt.Equal(start) || o.CreatedAt.After(start)) &&
               (o.CreatedAt.Equal(end) || o.CreatedAt.Before(end)) {
                orders = append(orders, o)
            }
        }
        return orders, nil
    }
}
