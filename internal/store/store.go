package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store map[string]entry

type entry struct {
	Value     int64
	UpdatedAt time.Time
}

func New() *Store {
	return &Store{}
}

func (s *Store) Set(key string, val int64) {
	(*s)[key] = entry{
		Value:     val,
		UpdatedAt: time.Now(),
	}
}

func (s *Store) Get(key string) (int64, bool) {
	val, exist := (*s)[key]
	return val.Value, exist
}

func (s *Store) ToFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("can not create directory: '%s': %s", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(absPath, data, 0644)
}

func FromFile(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	store := Store{}
	err = json.Unmarshal(data, &store)
	if err != nil {
		return nil, err
	}

	return &store, nil
}

func (s *Store) RemoveOldEntries(age time.Duration) int {
	now := time.Now()
	var cnt int

	for k, v := range *s {
		if now.Sub(v.UpdatedAt) > age {
			delete(*s, k)
			cnt++
		}
	}

	return cnt
}
