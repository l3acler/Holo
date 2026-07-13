package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// withCORS is a middleware that adds CORS headers to the response and handles OPTIONS requests.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// withLogger logs incoming requests and their status codes.
func withLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("[%s] %s - %d %s", r.Method, r.URL.Path, rw.status, http.StatusText(rw.status))
	})
}


func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// handleCollection manages GET and POST requests for a resource collection.
func handleCollection(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := r.PathValue("resource")

		switch r.Method {
		case http.MethodGet:
			items, exists := store.GetCollection(resource)
			if !exists {
				// Return an empty array if resource doesn't exist
				writeJSON(w, http.StatusOK, []map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, items)

		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Bad Request: Malformed JSON", http.StatusBadRequest)
				return
			}

			createdItem, err := store.CreateItem(resource, body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusCreated, createdItem)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleItem manages GET, PUT, PATCH, and DELETE requests for a specific item.
func handleItem(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := r.PathValue("resource")
		id := r.PathValue("id")

		switch r.Method {
		case http.MethodGet:
			item, exists := store.GetItem(resource, id)
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, item)

		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Bad Request: Malformed JSON", http.StatusBadRequest)
				return
			}

			updatedItem, exists, err := store.ReplaceItem(resource, id, body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, updatedItem)

		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Bad Request: Malformed JSON", http.StatusBadRequest)
				return
			}

			updatedItem, exists, err := store.UpdateItem(resource, id, body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, updatedItem)

		case http.MethodDelete:
			exists, err := store.DeleteItem(resource, id)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
