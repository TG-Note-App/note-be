package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Note - represent note entity
type Note struct {
	ID           int       `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	LastModified time.Time `json:"lastModified"`
	IsPinned     bool      `json:"isPinned"`
}

var db *sql.DB

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err = sql.Open("postgres", os.Getenv("PG_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	r := mux.NewRouter()

	r.HandleFunc("/notes", getNotes).Methods("GET")
	r.HandleFunc("/notes/{id}", getNoteByID).Methods("GET")
	r.HandleFunc("/notes", createNote).Methods("POST")
	r.HandleFunc("/notes/{id}", updateNote).Methods("PUT")
	r.HandleFunc("/notes/{id}", deleteNote).Methods("DELETE")
	r.HandleFunc("/notes/{id}/toggle-pin", togglePinNote).Methods("PUT")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/dist")))

	log.Println("Server started on :8080")

	// Add CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      corsMiddleware(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

// Toggle pin status of a note
func togglePinNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var body struct {
		IsPinned bool `json:"isPinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Toggle the pin status
	_, err := db.Exec("UPDATE notes SET is_pin = $1 WHERE id = $2", body.IsPinned, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getNotes(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query("SELECT id, user_id, title, content, last_modified, is_pin FROM notes")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.LastModified, &n.IsPinned); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		notes = append(notes, n)
	}

	err = json.NewEncoder(w).Encode(notes)
	if err != nil {
		log.Printf("getNotes: %s\n", err.Error())
	}
}

func getNoteByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var note Note
	err := db.QueryRow("SELECT id, user_id, title, content, last_modified, is_pin FROM notes WHERE id = $1", id).Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.LastModified, &note.IsPinned)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(note)
	if err != nil {
		log.Printf("getNote: %s\n", err.Error())
	}
}

func createNote(w http.ResponseWriter, r *http.Request) {
	var n Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO notes (user_id, title, content, last_modified, is_pin) VALUES ($1, $2, $3, $4, $5)", n.UserID, n.Title, n.Content, time.Now(), n.IsPinned)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func updateNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var n Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE notes SET title=$1, content=$2, last_modified=$3, is_pin=$4 WHERE id=$5", n.Title, n.Content, time.Now(), n.IsPinned, vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, err := db.Exec("DELETE FROM notes WHERE id=$1", vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
