package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestServer(t *testing.T) (*Store, *http.ServeMux, func()) {
	tmpFile, err := os.CreateTemp("", "db_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Write([]byte(`{"users": [{"id": "1", "name": "Bader"}]}`))
	tmpFile.Close()

	store, err := NewStore(tmpFile.Name(), false)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/{resource}", handleCollection(store))
	mux.HandleFunc("/{resource}/{id}", handleItem(store))

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return store, mux, cleanup
}

func TestCollectionGet(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(res) != 1 || res[0]["name"] != "Bader" {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestCollectionPost(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	body := []byte(`{"name": "Alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var res map[string]any
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res["name"] != "Alice" {
		t.Errorf("handler returned unexpected name: got %v", res["name"])
	}
	if res["id"] == nil || res["id"] == "" {
		t.Errorf("expected auto-generated ID, got %v", res["id"])
	}
}

func TestItemGet(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res map[string]any
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res["name"] != "Bader" {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestItemPut(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	// PUT completely replaces the object
	body := []byte(`{"role": "admin"}`)
	req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res map[string]any
	json.Unmarshal(rr.Body.Bytes(), &res)

	// 'name' should be gone, 'role' should be there, 'id' preserved
	if res["name"] != nil {
		t.Errorf("PUT should have replaced the object, but name is still present: %v", res)
	}
	if res["role"] != "admin" || res["id"] != "1" {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestItemPatch(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	// PATCH partially updates
	body := []byte(`{"role": "admin"}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/1", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res map[string]any
	json.Unmarshal(rr.Body.Bytes(), &res)

	// 'name' should still be there, 'role' should be added
	if res["name"] != "Bader" || res["role"] != "admin" {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestItemDelete(t *testing.T) {
	_, mux, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %v", status)
	}
}
