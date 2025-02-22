package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Note - represent note entity
type Note struct {
	ID           int       `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	LastModified time.Time `json:"lastModified"`
	IsPinned     bool      `json:"isPinned"`
	Files        []File    `json:"attachments"`
}

// File - represent file entity
type File struct {
	ID        int    `json:"id"`
	NoteID    int    `json:"noteId"`
	FileName  string `json:"filename"`
	Size      int    `json:"size"`
	Extension string `json:"extension"`
	URL       string `json:"url"`
}

var (
	db          *sql.DB
	minioClient *minio.Client
)

const (
	noteFilesBucket = "notes-files"
)

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

	// Initialize MinIO client
	if err := initMinioClient(); err != nil {
		log.Fatal("Error initializing MinIO client:", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/notes", getNotes).Methods("GET")
	r.HandleFunc("/notes/{id}", getNoteByID).Methods("GET")
	r.HandleFunc("/notes", createNote).Methods("POST")
	r.HandleFunc("/notes/{id}", updateNote).Methods("PUT")
	r.HandleFunc("/notes/{id}", deleteNote).Methods("DELETE")
	r.HandleFunc("/notes/{id}/toggle-pin", togglePinNote).Methods("PUT")
	r.HandleFunc("/notes/{id}/upload-file", uploadFile).Methods("POST")
	r.HandleFunc("/notes/{id}/delete-file", deleteFile).Methods("DELETE")

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

func initMinioClient() error {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return err
	}

	minioClient = client
	return nil
}

// Helper function to upload file to MinIO
func uploadFileToMinio(bucketName, objectName string, fileData []byte) (string, error) {
	// Check if bucket exists, create if it doesn't
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		return "", err
	}

	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return "", err
		}
	}

	// Upload the file
	reader := bytes.NewReader(fileData)
	_, err = minioClient.PutObject(context.Background(), bucketName, objectName, reader, int64(len(fileData)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", err
	}

	// Generate presigned URL for downloading
	// Set URL expiry to 7 days (or adjust as needed)
	reqParams := make(url.Values)
	presignedURL, err := minioClient.PresignedGetObject(context.Background(), bucketName, objectName, time.Hour*24*7, reqParams)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

// Helper function to delete file from MinIO
func deleteFileFromMinio(bucketName, objectName string) error {
	return minioClient.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
}

// Toggle pin status of a note
func togglePinNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Printf("Toggling pin status for note ID: %s", id)

	var body struct {
		IsPinned bool `json:"isPinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("Error decoding toggle pin request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Toggle the pin status
	_, err := db.Exec("UPDATE notes SET is_pin = $1 WHERE id = $2", body.IsPinned, id)
	if err != nil {
		log.Printf("Error updating pin status: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully toggled pin status to %v for note ID: %s", body.IsPinned, id)
	w.WriteHeader(http.StatusOK)
}

func getNotes(w http.ResponseWriter, _ *http.Request) {
	log.Println("Fetching all notes")
	// ... existing code ...

	// First get all notes
	rows, err := db.Query("SELECT id, user_id, title, content, last_modified, is_pin FROM notes")
	if err != nil {
		log.Printf("Error querying notes: %v", err)
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

		// Get files for this note
		fileRows, err := db.Query("SELECT id, note_id, file_name, size, ext, file_url FROM note_files WHERE note_id = $1", n.ID)
		if err != nil {
			log.Printf("Error querying files for note %d: %v", n.ID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = fileRows.Close() }()

		// Collect all files for this note
		var files []File
		for fileRows.Next() {
			var f File
			if err := fileRows.Scan(&f.ID, &f.NoteID, &f.FileName, &f.Size, &f.Extension, &f.URL); err != nil {
				log.Printf("Error scanning file for note %d: %v", n.ID, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("File: %+v", f)
			files = append(files, f)
		}

		n.Files = files
		notes = append(notes, n)
	}

	// ... existing code ...
	err = json.NewEncoder(w).Encode(notes)
	if err != nil {
		log.Printf("Error encoding notes response: %v", err)
	}
	log.Printf("Successfully retrieved %d notes", len(notes))
}

func getNoteByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Printf("Fetching note with ID: %s", id)

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
		log.Printf("Error encoding note response: %v", err)
	}
	log.Printf("Successfully retrieved note with ID: %s", id)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	log.Println("Creating new note")
	var n Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		log.Printf("Error decoding create note request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO notes (user_id, title, content, last_modified, is_pin) VALUES ($1, $2, $3, $4, $5)", n.UserID, n.Title, n.Content, time.Now(), n.IsPinned)
	if err != nil {
		log.Printf("Error creating note: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully created note for user: %s", n.UserID)
	w.WriteHeader(http.StatusCreated)
}

func updateNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Printf("Updating note with ID: %s", id)

	var n Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		log.Printf("Error decoding update note request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE notes SET title=$1, content=$2, last_modified=$3, is_pin=$4 WHERE id=$5", n.Title, n.Content, time.Now(), n.IsPinned, id)
	if err != nil {
		log.Printf("Error updating note: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully updated note with ID: %s", id)
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Printf("Deleting note with ID: %s", id)

	_, err := db.Exec("DELETE FROM notes WHERE id=$1", id)
	if err != nil {
		log.Printf("Error deleting note: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully deleted note with ID: %s", id)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	noteID := vars["id"]
	log.Printf("[uploadFile] Starting file upload for note ID: %s", noteID)

	// Parse multipart form with 32MB max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("[uploadFile] Error parsing multipart form: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[uploadFile] Error getting file from form: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

	log.Printf("[uploadFile] Received file: %s, size: %d bytes", header.Filename, header.Size)

	// Read file data
	fileData := make([]byte, header.Size)
	_, err = file.Read(fileData)
	if err != nil {
		log.Printf("[uploadFile] Error reading file data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[uploadFile] Successfully read file data")

	// Upload to MinIO
	bucketName := noteFilesBucket
	objectName := fmt.Sprintf("%s-%s", noteID, header.Filename)
	log.Printf("[uploadFile] Attempting to upload file to MinIO bucket: %s, object: %s", bucketName, objectName)

	downloadURL, err := uploadFileToMinio(bucketName, objectName, fileData)
	if err != nil {
		log.Printf("[uploadFile] Error uploading to MinIO: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[uploadFile] Download URL: %s", downloadURL)

	log.Printf("[uploadFile] Successfully uploaded file to MinIO")

	name, ext := getFileInfo(header.Filename)

	// Save file metadata to database with presigned URL
	log.Printf("[uploadFile] Saving file metadata to database")
	var fileID int
	err = db.QueryRow(
		"INSERT INTO note_files (note_id, file_name, size, ext, file_url) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		noteID, name, header.Size, ext, downloadURL,
	).Scan(&fileID)
	if err != nil {
		log.Printf("[uploadFile] Error saving file metadata: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[uploadFile] Successfully saved file metadata with ID: %d", fileID)

	// Return the file information
	fileInfo := File{
		ID:       fileID,
		NoteID:   parseInt(noteID),
		FileName: header.Filename,
		Size:     int(header.Size),
		URL:      downloadURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileInfo); err != nil {
		log.Printf("[uploadFile] Error encoding response: %v", err)
	}
	log.Printf("[uploadFile] Successfully completed file upload process for %s (ID: %d) in note ID: %s", header.Filename, fileID, noteID)
}

func getFileInfo(filename string) (string, string) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return name, ext
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	noteID := vars["id"]

	// Create a struct to hold the request body
	var requestBody struct {
		FileID int `json:"attachmentId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.Printf("Error decoding file ID: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := requestBody.FileID
	// Get file information from database
	var fileName string
	log.Printf("Deleting file with ID: %d from note ID: %s", fileID, noteID)
	err := db.QueryRow("SELECT file_name FROM note_files WHERE id = $1 AND note_id = $2", fileID, noteID).Scan(&fileName)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		log.Printf("Error querying file: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete from MinIO
	bucketName := noteFilesBucket
	objectName := fmt.Sprintf("%s-%s", noteID, fileName)
	if err := deleteFileFromMinio(bucketName, objectName); err != nil {
		log.Printf("Error deleting from MinIO: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete from database
	_, err = db.Exec("DELETE FROM note_files WHERE id = $1 AND note_id = $2", fileID, noteID)
	if err != nil {
		log.Printf("Error deleting file metadata: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted file ID %d from note ID: %s", fileID, noteID)
	w.WriteHeader(http.StatusOK)
}

// Helper function to parse string to int
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
