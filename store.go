package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store represents the in-memory data store with thread-safe access and file persistence.
type Store struct {
	mu         sync.RWMutex
	data       map[string][]map[string]any
	filePath   string
	memoryOnly bool
}

// NewStore creates a new store and loads initial data if the file exists.
func NewStore(filePath string, memoryOnly bool) (*Store, error) {
	s := &Store{
		data:       make(map[string][]map[string]any),
		filePath:   filePath,
		memoryOnly: memoryOnly,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// load reads the JSON file into memory. If the file doesn't exist, it starts empty.
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Automatically create with {}
			err = os.WriteFile(s.filePath, []byte("{}"), 0644)
			if err != nil {
				return fmt.Errorf("failed to create initial json file: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Consider files with only whitespace as empty
	isEmpty := true
	for _, char := range b {
		if char != ' ' && char != '\t' && char != '\n' && char != '\r' {
			isEmpty = false
			break
		}
	}

	if isEmpty {
		// Initialize empty file with {}
		err = os.WriteFile(s.filePath, []byte("{}"), 0644)
		if err != nil {
			return fmt.Errorf("failed to initialize empty json file: %w", err)
		}
		return nil
	}

	if err := json.Unmarshal(b, &s.data); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// saveToFile writes the current in-memory state back to the JSON file atomically.
// It assumes the caller already holds a lock (either read or write).
func (s *Store) saveToFile() error {
	if s.memoryOnly {
		return nil
	}

	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Atomic write: write to temp file, then rename
	dir := filepath.Dir(s.filePath)
	tmpFile, err := os.CreateTemp(dir, "holo-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(b); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.filePath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetCollection returns all items for a given resource.
func (s *Store) GetCollection(resource string) ([]map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, exists := s.data[resource]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external mutation
	itemsCopy := make([]map[string]any, len(items))
	for i, item := range items {
		itemCopy := make(map[string]any)
		for k, v := range item {
			itemCopy[k] = v
		}
		itemsCopy[i] = itemCopy
	}

	return itemsCopy, true
}

// GetItem returns a single item by resource and ID.
func (s *Store) GetItem(resource, id string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, exists := s.data[resource]
	if !exists {
		return nil, false
	}

	for _, item := range items {
		// Normalize map ID to string to match the requested string ID
		if mapID, ok := item["id"]; ok && fmt.Sprintf("%v", mapID) == id {
			// Return a copy
			itemCopy := make(map[string]any)
			for k, v := range item {
				itemCopy[k] = v
			}
			return itemCopy, true
		}
	}

	return nil, false
}

// CreateItem adds a new item to the resource collection.
func (s *Store) CreateItem(resource string, item map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := item["id"]; !ok {
		item["id"] = generateID()
	}

	s.data[resource] = append(s.data[resource], item)

	if err := s.saveToFile(); err != nil {
		s.data[resource] = s.data[resource][:len(s.data[resource])-1]
		return nil, err
	}

	return item, nil
}

// ReplaceItem replaces an entire item (PUT semantics). ID is preserved.
func (s *Store) ReplaceItem(resource, id string, newItem map[string]any) (map[string]any, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, exists := s.data[resource]
	if !exists {
		return nil, false, nil
	}

	for i, item := range items {
		if mapID, ok := item["id"]; ok && fmt.Sprintf("%v", mapID) == id {
			// Ensure ID is preserved
			newItem["id"] = item["id"]

			originalItem := items[i]
			items[i] = newItem

			if err := s.saveToFile(); err != nil {
				items[i] = originalItem
				return nil, true, err
			}

			updatedCopy := make(map[string]any)
			for k, v := range newItem {
				updatedCopy[k] = v
			}
			return updatedCopy, true, nil
		}
	}

	return nil, false, nil
}

// UpdateItem partially updates an item (PATCH semantics).
func (s *Store) UpdateItem(resource, id string, updates map[string]any) (map[string]any, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, exists := s.data[resource]
	if !exists {
		return nil, false, nil
	}

	for i, item := range items {
		if mapID, ok := item["id"]; ok && fmt.Sprintf("%v", mapID) == id {
			// Create a copy of the original for rollback
			originalItem := make(map[string]any)
			for k, v := range item {
				originalItem[k] = v
			}

			// Apply updates
			for k, v := range updates {
				if k != "id" { // Do not allow updating the ID
					item[k] = v
				}
			}

			if err := s.saveToFile(); err != nil {
				items[i] = originalItem
				return nil, true, err
			}

			updatedCopy := make(map[string]any)
			for k, v := range item {
				updatedCopy[k] = v
			}
			return updatedCopy, true, nil
		}
	}

	return nil, false, nil
}

// DeleteItem removes an item from the resource collection.
func (s *Store) DeleteItem(resource, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, exists := s.data[resource]
	if !exists {
		return false, nil
	}

	for i, item := range items {
		if mapID, ok := item["id"]; ok && fmt.Sprintf("%v", mapID) == id {
			originalItems := make([]map[string]any, len(items))
			copy(originalItems, items)

			s.data[resource] = append(items[:i], items[i+1:]...)

			if err := s.saveToFile(); err != nil {
				s.data[resource] = originalItems
				return true, err
			}

			return true, nil
		}
	}

	return false, nil
}
