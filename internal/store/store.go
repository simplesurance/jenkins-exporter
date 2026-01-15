// Package store implements a persistent file based store for the last processed build IDs.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const version = 1

// ErrIncompatibleFormat is returned if the data store file format is not compatible.
var ErrIncompatibleFormat = errors.New("incompatible format")

// Store is the data structure for the state store.
type Store struct {
	Builds  map[string]*Build
	Version uint64
}

// Build holds the state of a build.
type Build struct {
	HighestRecordedBuildID int64
	UnrecordedBuildIDs     map[int64]time.Time
	UpdatedAt              time.Time
}

// SetHighestRecordedBuildID sets the highest recorded build ID and updates the UpdatedAt timestamp.
func (b *Build) SetHighestRecordedBuildID(v int64) {
	b.HighestRecordedBuildID = v
	b.UpdatedAt = time.Now()
}

// UnrecordedBuildIDsIsEmpty returns true if the list of unrecorded build IDs is empty.
func (b *Build) UnrecordedBuildIDsIsEmpty() bool {
	return len(b.UnrecordedBuildIDs) == 0
}

// IsUnrecorded returns true if the build ID is in the unrecorded build map.
func (b *Build) IsUnrecorded(id int64) bool {
	_, exists := b.UnrecordedBuildIDs[id]
	return exists
}

// DeleteUnrecorded removes a build ID from the unrecorded build map and updates the UpdatedAt timestamp.
func (b *Build) DeleteUnrecorded(id int64) {
	delete(b.UnrecordedBuildIDs, id)
	b.UpdatedAt = time.Now()
}

// AddUnrecorded adds a build ID to the unrecorded build map and updates the UpdatedAt timestamp.
func (b *Build) AddUnrecorded(id int64) (added bool) {
	if b.IsUnrecorded(id) {
		return false
	}

	b.UnrecordedBuildIDs[id] = time.Now()
	b.UpdatedAt = time.Now()
	return true
}

func newBuild(highestBuildID int64) *Build {
	return &Build{
		HighestRecordedBuildID: highestBuildID,
		UnrecordedBuildIDs:     map[int64]time.Time{},
		UpdatedAt:              time.Now(),
	}
}

// New returns a new empty store.
func New() *Store {
	return &Store{
		Builds:  map[string]*Build{},
		Version: version,
	}
}

// NewBuild creates a new build, adds it to the store and returns it.
func (s *Store) NewBuild(key string, highestRecordedBuildID int64) *Build {
	b := newBuild(highestRecordedBuildID)
	s.Builds[key] = b
	return b
}

// Get returns a build from the store.
func (s *Store) Get(key string) *Build {
	return s.Builds[key]
}

// ToFile serializes the store to a file.
func (s *Store) ToFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("can not create directory: %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(absPath, data, 0644)
}

// FromFile deserializes a store from a file.
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

	if store.Version == 0 {
		return nil, ErrIncompatibleFormat
	}

	return &store, nil
}

// RemoveOldBuilds removes all builds from the store that are older than maxAge.
func (s *Store) RemoveOldBuilds(maxAge time.Duration) int {
	now := time.Now()
	var cnt int

	for k, build := range s.Builds {
		if now.Sub(build.UpdatedAt) > maxAge {
			delete(s.Builds, k)
			cnt++
		}
	}

	return cnt
}

// RemoveOldUnrecordedEntries removes all unrecorded build entries from the store that are older than maxAge.
func (s *Store) RemoveOldUnrecordedEntries(maxAge time.Duration) int {
	now := time.Now()
	var cnt int

	for _, build := range s.Builds {
		for buildID, updatedAt := range build.UnrecordedBuildIDs {
			if now.Sub(updatedAt) > maxAge {
				delete(build.UnrecordedBuildIDs, buildID)
				cnt++
			}
		}
	}

	return cnt
}
