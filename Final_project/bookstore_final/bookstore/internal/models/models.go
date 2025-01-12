	// internal/models/models.gopackage models
	package models
	import (
		"time"
	)

	type Book struct {
		ID          int       `json:"id"`
		Title       string    `json:"title"`
		Author      Author    `json:"author"`
		Genres      []string  `json:"genres"`
		PublishedAt time.Time `json:"published_at"`
		Price       float64   `json:"price"`
		Stock       int       `json:"stock"`
	}

	type Author struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Bio       string `json:"bio"`
	}

	type Customer struct {
		ID        int       `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Address   Address   `json:"address"`
		CreatedAt time.Time `json:"created_at"`
	}

	type Address struct {
		Street     string `json:"street"`
		City       string `json:"city"`
		State      string `json:"state"`
		PostalCode string `json:"postal_code"`
		Country    string `json:"country"`
	}

	type Order struct {
		ID         int         `json:"id"`
		Customer   Customer    `json:"customer"`
		Items      []OrderItem `json:"items"`
		TotalPrice float64     `json:"total_price"`
		CreatedAt  time.Time   `json:"created_at"`
		Status     string      `json:"status"`
	}

	type OrderItem struct {
		Book     Book `json:"book"`
		Quantity int  `json:"quantity"`
	}

	type SalesReport struct {
		Timestamp     time.Time   `json:"timestamp"`
		TotalRevenue  float64     `json:"total_revenue"`
		TotalOrders   int         `json:"total_orders"`
		TopSellingBooks []BookSales `json:"top_selling_books"`
	}

	type BookSales struct {
		Book     Book `json:"book"`
		Quantity int  `json:"quantity_sold"`
	}


	type SearchCriteria struct {
		Title  string   `json:"title"`
		Author string   `json:"author"`
		Genres []string `json:"genres"`
		MinPrice float64 `json:"min_price"`
		MaxPrice float64 `json:"max_price"`
	}

	type ErrorResponse struct {
		Error string `json:"error"`
	}