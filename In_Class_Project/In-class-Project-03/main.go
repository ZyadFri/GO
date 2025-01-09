package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

type Course struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Credit int    `json:"credit"`
}

type Student struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Major   string   `json:"major"`
	Address Address  `json:"address"`
	Courses []Course `json:"courses"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type StudentRepository interface {
	Create(student *Student) error
	GetByID(id int64) (*Student, error)
	Update(id int64, student *Student) error
	Delete(id int64) error
	List() ([]*Student, error)
}

type InMemoryStudentRepository struct {
	mutex    sync.RWMutex
	students map[int64]*Student
	nextID   int64
}

func NewInMemoryStudentRepository() *InMemoryStudentRepository {
	return &InMemoryStudentRepository{
		students: make(map[int64]*Student),
		nextID:   1,
	}
}

func (r *InMemoryStudentRepository) Create(student *Student) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	student.ID = r.nextID
	r.students[student.ID] = student
	r.nextID++
	return nil
}

func (r *InMemoryStudentRepository) GetByID(id int64) (*Student, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	student, exists := r.students[id]
	if !exists {
		return nil, fmt.Errorf("student not found")
	}
	return student, nil
}

func (r *InMemoryStudentRepository) Update(id int64, student *Student) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.students[id]; !exists {
		return fmt.Errorf("student not found")
	}

	student.ID = id
	r.students[id] = student
	return nil
}

func (r *InMemoryStudentRepository) Delete(id int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.students[id]; !exists {
		return fmt.Errorf("student not found")
	}

	delete(r.students, id)
	return nil
}

func (r *InMemoryStudentRepository) List() ([]*Student, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	students := make([]*Student, 0, len(r.students))
	for _, student := range r.students {
		students = append(students, student)
	}
	return students, nil
}

type StudentHandler struct {
	repo StudentRepository
}

func NewStudentHandler(repo StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func (h *StudentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/students":
		h.CreateStudent(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/students":
		h.ListStudents(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/students/"):
		h.GetStudent(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/students/"):
		h.UpdateStudent(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/students/"):
		h.DeleteStudent(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *StudentHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(&student); err != nil {
		h.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

func (h *StudentHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	id, err := h.getIDFromPath(r.URL.Path)
	if err != nil {
		h.sendError(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	student, err := h.repo.GetByID(id)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(student)
}

func (h *StudentHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	id, err := h.getIDFromPath(r.URL.Path)
	if err != nil {
		h.sendError(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Update(id, &student); err != nil {
		h.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(student)
}

func (h *StudentHandler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	id, err := h.getIDFromPath(r.URL.Path)
	if err != nil {
		h.sendError(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		h.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *StudentHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	students, err := h.repo.List()
	if err != nil {
		h.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(students)
}

func (h *StudentHandler) getIDFromPath(path string) (int64, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid path")
	}

	return strconv.ParseInt(parts[2], 10, 64)
}

func (h *StudentHandler) sendError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func main() {
	repo := NewInMemoryStudentRepository()
	handler := NewStudentHandler(repo)

	http.Handle("/students", handler)
	http.Handle("/students/", handler)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
