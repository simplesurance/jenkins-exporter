package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Store map[string]int64

func New() *Store {
	return &Store{}
}

func (s *Store) Set(key string, val int64) {
	(*s)[key] = val
}

func (s *Store) Get(key string) (int64, bool) {
	val, exist := (*s)[key]
	return val, exist
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

	return ioutil.WriteFile(absPath, data, 0644)
}

func FromFile(path string) (*Store, error) {
	data, err := ioutil.ReadFile(path)
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
