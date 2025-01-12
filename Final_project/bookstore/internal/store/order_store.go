
package store

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "bookstore/internal/interfaces"
    "bookstore/internal/models"
)

type PostgresOrderStore struct {
    db *sql.DB
}

func NewPostgresOrderStore(db *sql.DB) (interfaces.OrderStore, error) {
    return &PostgresOrderStore{db: db}, nil
}

func (s *PostgresOrderStore) CreateOrder(ctx context.Context, order models.Order) (models.Order, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return order, err
    }

 
    var total float64
    for _, item := range order.Items {
        total += item.Book.Price * float64(item.Quantity)
    }
    now := time.Now()


    query := `
        INSERT INTO orders (customer_id, total_price, created_at, status)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
    err = tx.QueryRowContext(ctx, query,
        order.Customer.ID,
        total,
        now,
        "pending",
    ).Scan(&order.ID)
    if err != nil {
        tx.Rollback()
        return order, fmt.Errorf("CreateOrder (orders insert): %w", err)
    }
    order.CreatedAt = now
    order.TotalPrice = total
    order.Status = "pending"


    for _, item := range order.Items {
        ins := `
            INSERT INTO order_items (order_id, book_id, quantity)
            VALUES ($1, $2, $3)
        `
        _, err := tx.ExecContext(ctx, ins, order.ID, item.Book.ID, item.Quantity)
        if err != nil {
            tx.Rollback()
            return order, fmt.Errorf("CreateOrder (order_items insert): %w", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return order, err
    }
    return order, nil
}

func (s *PostgresOrderStore) GetOrder(ctx context.Context, id int) (models.Order, error) {
    var order models.Order


    query := `
        SELECT o.id, o.customer_id, o.total_price, o.created_at, o.status,
               c.id, c.name, c.email, c.street, c.city, c.state, c.postal_code, c.country, c.created_at
        FROM orders o
        JOIN customers c ON o.customer_id = c.id
        WHERE o.id = $1
    `
    row := s.db.QueryRowContext(ctx, query, id)

    var cust models.Customer
    err := row.Scan(
        &order.ID,
        &order.Customer.ID,
        &order.TotalPrice,
        &order.CreatedAt,
        &order.Status,
        &cust.ID,
        &cust.Name,
        &cust.Email,
        &cust.Address.Street,
        &cust.Address.City,
        &cust.Address.State,
        &cust.Address.PostalCode,
        &cust.Address.Country,
        &cust.CreatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return order, fmt.Errorf("order not found with id: %d", id)
        }
        return order, err
    }
    order.Customer = cust


    items, err := s.getOrderItems(ctx, order.ID)
    if err != nil {
        return order, err
    }
    order.Items = items

    return order, nil
}

func (s *PostgresOrderStore) UpdateOrder(ctx context.Context, id int, updated models.Order) (models.Order, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return updated, err
    }
    defer tx.Rollback()


    var total float64
    for _, item := range updated.Items {
        total += item.Book.Price * float64(item.Quantity)
    }


    up := `
        UPDATE orders
        SET customer_id = $1, total_price = $2, status = $3
        WHERE id = $4
    `
    _, err = tx.ExecContext(ctx, up,
        updated.Customer.ID,
        total,
        updated.Status,
        id,
    )
    if err != nil {
        return updated, fmt.Errorf("UpdateOrder (orders update): %w", err)
    }

    existing, err := s.GetOrder(ctx, id)
    if err != nil {
        return updated, fmt.Errorf("UpdateOrder: cannot fetch existing order: %w", err)
    }


    del := `DELETE FROM order_items WHERE order_id = $1`
    _, err = tx.ExecContext(ctx, del, id)
    if err != nil {
        return updated, fmt.Errorf("UpdateOrder (delete items): %w", err)
    }


    for _, item := range updated.Items {
        ins := `
            INSERT INTO order_items (order_id, book_id, quantity)
            VALUES ($1, $2, $3)
        `
        _, err := tx.ExecContext(ctx, ins, id, item.Book.ID, item.Quantity)
        if err != nil {
            return updated, fmt.Errorf("UpdateOrder (insert items): %w", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return updated, err
    }

    updated.ID = id
    updated.CreatedAt = existing.CreatedAt
    updated.TotalPrice = total
    return updated, nil
}

func (s *PostgresOrderStore) DeleteOrder(ctx context.Context, id int) error {
    query := `DELETE FROM orders WHERE id = $1`
    _, err := s.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("DeleteOrder error: %w", err)
    }
    return nil
}

func (s *PostgresOrderStore) ListOrders(ctx context.Context) ([]models.Order, error) {
    query := `
        SELECT o.id, o.customer_id, o.total_price, o.created_at, o.status,
               c.id, c.name, c.email, c.street, c.city, c.state, c.postal_code, c.country, c.created_at
        FROM orders o
        JOIN customers c ON o.customer_id = c.id
        ORDER BY o.id
    `
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("ListOrders error: %w", err)
    }
    defer rows.Close()

    var results []models.Order
    for rows.Next() {
        var order models.Order
        var cust models.Customer
        if err := rows.Scan(
            &order.ID,
            &order.Customer.ID,
            &order.TotalPrice,
            &order.CreatedAt,
            &order.Status,
            &cust.ID,
            &cust.Name,
            &cust.Email,
            &cust.Address.Street,
            &cust.Address.City,
            &cust.Address.State,
            &cust.Address.PostalCode,
            &cust.Address.Country,
            &cust.CreatedAt,
        ); err != nil {
            return nil, err
        }
        order.Customer = cust


        items, err := s.getOrderItems(ctx, order.ID)
        if err != nil {
            return nil, err
        }
        order.Items = items

        results = append(results, order)
    }
    if err = rows.Err(); err != nil {
        return nil, err
    }
    return results, nil
}

func (s *PostgresOrderStore) GetOrdersInTimeRange(ctx context.Context, start, end time.Time) ([]models.Order, error) {
    query := `
        SELECT id
        FROM orders
        WHERE created_at BETWEEN $1 AND $2
        ORDER BY id
    `
    rows, err := s.db.QueryContext(ctx, query, start, end)
    if err != nil {
        return nil, fmt.Errorf("GetOrdersInTimeRange error: %w", err)
    }
    defer rows.Close()

    var orders []models.Order
    for rows.Next() {
        var oid int
        if err := rows.Scan(&oid); err != nil {
            return nil, err
        }
        o, err := s.GetOrder(ctx, oid)
        if err != nil {
            return nil, err
        }
        orders = append(orders, o)
    }
    return orders, rows.Err()
}


func (s *PostgresOrderStore) getOrderItems(ctx context.Context, orderID int) ([]models.OrderItem, error) {
    query := `
        SELECT oi.book_id, oi.quantity,
               b.title, b.price, b.stock,
               a.id, a.first_name, a.last_name, a.bio
        FROM order_items oi
        JOIN books b ON oi.book_id = b.id
        JOIN authors a ON b.author_id = a.id
        WHERE oi.order_id = $1
    `
    rows, err := s.db.QueryContext(ctx, query, orderID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []models.OrderItem
    for rows.Next() {
        var (
            item   models.OrderItem
            book   models.Book
            author models.Author
        )
        err := rows.Scan(
            &book.ID,
            &item.Quantity,
            &book.Title,
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
        item.Book = book
        items = append(items, item)
    }
    return items, rows.Err()
}
