#!/usr/bin/env bash

# ===========================================================
# test_all.sh
#
# A script with many cURL tests for my Bookstore API.
# It tests JWT login, authors, books, orders, customers,
# advanced search, and more.
#

# Base URL for the API
BASE_URL="http://localhost:8080"

# We'll store the JWT token here once we get it
TOKEN=""

echo "============================="
echo "1) Ping the server (unprotected)"
echo "============================="
curl -i $BASE_URL/ping
echo -e "\n\n"

# -----------------------------------------------------------
# 2) LOGIN to get a JWT (username=admin, password=password)
# -----------------------------------------------------------
echo "============================="
echo "2) Login to retrieve JWT"
echo "============================="
LOGIN_RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  $BASE_URL/api/login)

echo "Login response: $LOGIN_RESPONSE"

# Extract token from JSON (simple parsing - not robust)
TOKEN=$(echo "$LOGIN_RESPONSE" | sed -E 's/.*"token":"([^"]+)".*/\1/')
echo "Token extracted: $TOKEN"
echo -e "\n\n"

# Helper function to send requests with JWT
function auth_curl() {
  METHOD=$1
  ENDPOINT=$2
  DATA=$3  # optional JSON data

  if [ -z "$DATA" ]; then
    curl -i -X $METHOD \
      -H "Authorization: Bearer $TOKEN" \
      "$BASE_URL$ENDPOINT"
  else
    curl -i -X $METHOD \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "$DATA" \
      "$BASE_URL$ENDPOINT"
  fi
  echo -e "\n\n"
}

# =========================================
# 3) TEST AUTHORS
# =========================================

echo "============================="
echo "3) Create a new author (POST /api/authors)"
echo "============================="
auth_curl "POST" "/api/authors" '{"first_name":"Jane","last_name":"Austen","bio":"English novelist"}'

echo "============================="
echo "List all authors (GET /api/authors)"
echo "============================="
auth_curl "GET" "/api/authors"

echo "============================="
echo "Create another author (POST /api/authors)"
echo "============================="
auth_curl "POST" "/api/authors" '{"first_name":"Isaac","last_name":"Asimov","bio":"Famous sci-fi author"}'

echo "List all authors again (GET /api/authors)"
auth_curl "GET" "/api/authors"

echo "============================="
echo "Get author by ID (GET /api/authors/1)"
echo "============================="
auth_curl "GET" "/api/authors/1"

echo "============================="
echo "Update author #1 (PUT /api/authors/1)"
echo "============================="
auth_curl "PUT" "/api/authors/1" '{"first_name":"Jane","last_name":"Austen","bio":"19th century English novelist"}'

echo "============================="
echo "Delete author #2 (DELETE /api/authors/2)"
echo "(WARNING: if the author is referenced by a book, might fail due to foreign key checks)"
echo "============================="
auth_curl "DELETE" "/api/authors/2"

# =========================================
# 4) TEST BOOKS
# =========================================

echo "============================="
echo "4) Create a new book (POST /api/books)"
echo "============================="
# We'll assume the author ID #1 still exists
auth_curl "POST" "/api/books" '{
  "title":"Pride and Prejudice",
  "author":{"id":1},
  "published_at":"1813-01-28T00:00:00Z",
  "price":9.99,
  "stock":100
}'

echo "List all books (GET /api/books)"
auth_curl "GET" "/api/books"

echo "============================="
echo "Create another book"
echo "============================="
auth_curl "POST" "/api/books" '{
  "title":"Emma",
  "author":{"id":1},
  "published_at":"1815-12-23T00:00:00Z",
  "price":10.99,
  "stock":50
}'

echo "List all books again"
auth_curl "GET" "/api/books"

echo "============================="
echo "Update book #1 (PUT /api/books/1)"
echo "============================="
auth_curl "PUT" "/api/books/1" '{
  "title":"Pride and Prejudice (Updated)",
  "author":{"id":1},
  "published_at":"1813-01-28T00:00:00Z",
  "price":11.99,
  "stock":200
}'

echo "============================="
echo "Get book #1 (GET /api/books/1)"
echo "============================="
auth_curl "GET" "/api/books/1"

echo "============================="
echo "Search for books with min_price=10, max_price=20, published_after=1814-01-01T00:00:00Z"
echo "============================="
auth_curl "GET" "/api/books?min_price=10&max_price=20&published_after=1814-01-01T00:00:00Z"

echo "============================="
echo "Delete book #1"
echo "============================="
auth_curl "DELETE" "/api/books/1"

# =========================================
# 5) TEST CUSTOMERS
# =========================================

echo "============================="
echo "5) Create a new customer (POST /api/customers)"
echo "============================="
auth_curl "POST" "/api/customers" '{
  "name":"John Doe",
  "email":"john.doe@example.com",
  "address":{
    "street":"123 Main St",
    "city":"Boston",
    "state":"MA",
    "postal_code":"02108",
    "country":"USA"
  }
}'

echo "List all customers (GET /api/customers)"
auth_curl "GET" "/api/customers"

echo "============================="
echo "Update customer #1 (PUT /api/customers/1)"
echo "============================="
auth_curl "PUT" "/api/customers/1" '{
  "name":"John Doe (Updated)",
  "email":"john.newemail@example.com",
  "address":{
    "street":"456 Park Ave",
    "city":"Boston",
    "state":"MA",
    "postal_code":"02109",
    "country":"USA"
  }
}'

echo "Get customer #1"
auth_curl "GET" "/api/customers/1"

# =========================================
# 6) TEST ORDERS
# =========================================

echo "============================="
echo "6) Create an order (POST /api/orders)"
echo "============================="
echo "(We assume you have some existing books with ID #2, etc.)"

auth_curl "POST" "/api/orders" '{
  "customer": {"id":1},
  "items": [
    {
      "book": {"id":2},
      "quantity": 2
    }
  ]
}'

echo "List all orders (GET /api/orders)"
auth_curl "GET" "/api/orders"

echo "============================="
echo "Update order #1 (PUT /api/orders/1)"
echo "(Add more items or adjust quantity, for example.)"
echo "============================="
auth_curl "PUT" "/api/orders/1" '{
  "customer": {"id":1},
  "items": [
    {
      "book": {"id":2},
      "quantity": 3
    }
  ],
  "status":"completed"
}'

echo "Get order #1"
auth_curl "GET" "/api/orders/1"

echo "============================="
echo "Delete order #1"
echo "============================="
auth_curl "DELETE" "/api/orders/1"

# =========================================
# 7) TEST REPORTS
# =========================================

echo "============================="
echo "7) Get sales reports (GET /api/reports/sales)"
echo "(We can specify ?start_date=2025-01-01&end_date=2025-12-31 if you like.)"
echo "============================="
auth_curl "GET" "/api/reports/sales?start_date=2025-01-01&end_date=2025-12-31"

echo "===== ALL TESTS COMPLETE ====="
echo "You may now review the above HTTP status codes and JSON responses."
