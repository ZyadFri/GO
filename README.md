# Bookstore API

A fully featured Go-based Bookstore API that uses **PostgreSQL** for persistence, **JWT** tokens for authentication, and supports **advanced searching**, **auto price adjustments**, and more. Originally, data was stored in JSON files but now is migrated to Postgres. This README explains every feature and how to run it step by step.

---

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
   1. [Persistence: JSON to PostgreSQL](#persistence-json-to-postgresql)
   2. [Authentication & JWT](#authentication--jwt)
   3. [Advanced Search & Filters](#advanced-search--filters)
   4. [Auto Price Adjustments (SalesReporter)](#auto-price-adjustments-salesreporter)
3. [Project Structure](#project-structure)
4. [Detailed Endpoints](#detailed-endpoints)
5. [How to Run](#how-to-run)
   1. [A) Start PostgreSQL (Docker or local)](#a-start-postgresql-docker-or-local)
   2. [B) Run Go Server](#b-run-go-server)
   3. [C) Confirm /ping](#c-confirm-ping)
   4. [D) Logging In (JWT)](#d-logging-in-jwt)
6. [Testing the App](#testing-the-app)
   1. [Option 1: `test_all.sh`](#option-1-test_allsh)
   2. [Option 2: Manual cURL/Postman Tests](#option-2-manual-curlpostman-tests)
7. [Enhancements & Admin Credentials](#enhancements--admin-credentials)
8. [Screenshots](#screenshots)
9. [License / Credits](#license--credits)

---

## Overview

This Bookstore API was initially **in-memory**, storing data in JSON files for authors, books, etc. We later **migrated** to **PostgreSQL** so the data is fully persistent. We also introduced:
- **JWT authentication** (login with `admin/password` to get a token).
- **Advanced searching** of books (by price range, published dates, stock).
- **A SalesReporter** that periodically generates sales reports and automatically adjusts the price of top-selling books by +10%.

The code is written in **Go (Golang)** with the following libraries:
- [Gorilla Mux](https://github.com/gorilla/mux) for routing
- [github.com/lib/pq](https://github.com/lib/pq) for Postgres
- [github.com/golang-jwt/jwt/v4](https://github.com/golang-jwt/jwt) for JWT tokens
- [github.com/rs/cors](https://github.com/rs/cors) for CORS handling

---

## Features

1. **CRUD Endpoints**  
   - Authors, Books, Customers, Orders, Reports  
   - Each resource has routes to create, read (list & single), update, and delete.

2. **Persistence: JSON to PostgreSQL**  
   - Originally, data was read from `.json` files (`authors.json`, `books.json`, etc.).  
   - Now, the app connects to **Postgres** (via `sql.DB`) and runs `CREATE TABLE` migrations automatically.  
   - The old JSON files are no longer needed—**you can safely remove them** once you’re sure you don’t need that data.  

3. **Authentication & JWT**  
   - There’s a `/api/login` endpoint. If you POST `{"username":"admin","password":"password"}`, you get a JWT token.  
   - All other routes under `/api` require `Authorization: Bearer <token>`.  
   - If you don’t provide a valid token, you get `401 Unauthorized`.  

4. **Advanced Search & Filters**  
   - **Books** can be filtered by `title`, `author`, `min_price`, `max_price`, `published_before`, `published_after`, `min_stock`, `max_stock`, etc.  
   - Use query parameters, e.g.:  
     ```
     GET /api/books?min_price=10&max_price=50&published_after=2021-01-01T00:00:00Z
     ```
   - The server builds a dynamic SQL `WHERE` clause to do the filtering.

5. **Auto Price Adjustments (SalesReporter)**  
   - The `SalesReporter` runs every `24 * time.Minute` by default (in `main.go`).  
   - It looks at orders from the last 24 hours, finds top 3 best-selling books, and **raises their price by 10%**.  
   - It also saves a JSON report in `output-reports/`.

---

## Project Structure
bookstore/
├── cmd/
│   └── server/
│       └── main.go        // The main entry point
├── internal/
│   ├── auth/              // JWT manager, middleware
│   ├── handlers/          // All HTTP handlers (author_handler.go, etc.)
│   ├── interfaces/        // Store interface definitions
│   ├── models/            // Data models (Book, Author, Customer, etc.)
│   ├── reports/           // SalesReporter logic
│   └── store/             // Postgres-based store implementations
├── pkg/
│   └── utils/             // logger.go, etc.
├── test_all.sh            // Large script with many cURL commands
├── go.mod
├── go.sum
├── README.md              // This file
└── ...

---

## Detailed Endpoints

- **Unprotected**:  
  - `GET /ping` → returns `"pong"`.  
  - `POST /api/login` → body: `{"username":"admin","password":"password"}`, returns JSON with `"token":"<JWT>"`.

- **Authors** (require JWT):
  - `POST /api/authors` → create new author  
  - `GET /api/authors` → list all authors  
  - `GET /api/authors/{id}` → get single author  
  - `PUT /api/authors/{id}` → update an author  
  - `DELETE /api/authors/{id}` → remove an author (checks foreign key usage)

- **Books** (JWT):
  - `POST /api/books` → create new book  
  - `GET /api/books` → list all or **search** with query params  
  - `GET /api/books/{id}` → single  
  - `PUT /api/books/{id}` → update  
  - `DELETE /api/books/{id}` → remove

- **Customers** (JWT):
  - `POST /api/customers`  
  - `GET /api/customers`  
  - `GET /api/customers/{id}`  
  - `PUT /api/customers/{id}`  
  - `DELETE /api/customers/{id}`

- **Orders** (JWT):
  - `POST /api/orders` → create an order (updates stock)  
  - `GET /api/orders` → list  
  - `GET /api/orders/{id}` → single  
  - `PUT /api/orders/{id}` → update items, recalc stock, total  
  - `DELETE /api/orders/{id}` → restore stock, then delete

- **Reports** (JWT):
  - `GET /api/reports/sales?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD` → returns sales data

---

## How to Run

Below are **very detailed** steps:

### A) Start PostgreSQL (Docker or local)

1. If you have a `docker-compose.yml` like:
   ```yaml
   version: '3.8'
   services:
     postgres:
       image: postgres:15
       container_name: bookstore-postgres
       environment:
         - POSTGRES_USER=postgres
         - POSTGRES_PASSWORD=secret
       ports:
         - "5432:5432"
1.  docker-compose up -d  //This spins up Postgres on port 5432 with user postgres and password secret.
    

**Alternatively**, install Postgres locally and create a bookstore database with a user/password matching the connection string in main.go (e.g. postgres://postgres:secret@localhost:5432/bookstore?sslmode=disable).

### B) Run Go Server

1. go mod tidy
2. go run cmd/server/main.go
3.Starting sales reporter...
4.Starting server on port 8080...  // That means the migrations ran, tables were created if needed, and now the server is listening on localhost:8080.
    

### C) Confirm /ping

Open a browser or cURL:

`   bash curl http://localhost:8080/ping   `

You should get back pong.

### D) Logging In (JWT)

1.  bash curl -X POST \\ -H "Content-Type: application/json" \\ -d '{"username":"admin","password":"password"}' \\ http://localhost:8080/api/login
    
2.  The response includes: {"token":""}.
    
    

Testing the App
---------------

### Option 1: test\_all.sh

We provide a script named test\_all.sh that runs **many** cURL requests in sequence:

1.  chmod +x test\_all.sh
    
2. ./test\_all.sh
    
3.  It will:
    
    *   Ping the server
        
    *   Login to get a JWT
        
    *   Create authors, books, customers, orders
        
    *   Update them
        
    *   Delete some resources
        
    *   Show advanced searches
        
    *   Finally, fetch reports.
        

If you’re on **Windows** without WSL, you can do:

`  bash bash test_all.sh   `

in **Git Bash**.

The script prints out each HTTP response. Look for 200 OK, 201 Created, etc.

### Option 2: Manual cURL/Postman Tests

*   **Login** to get your token.
    
*   **Hit** /api/books with GET, POST, etc.
    
*   **Use** Authorization: Bearer in each request.
    
*   bash curl -H "Authorization: Bearer " "http://localhost:8080/api/books?min\_stock=10&min\_price=5&published\_after=2022-01-01T00:00:00Z"
    


### Screenshots

## Ping (Unprotected)
![image](https://github.com/user-attachments/assets/56aecbc8-0ed5-42c9-9f82-b21bffecfbe6)



