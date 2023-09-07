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

var ErrIncompatibleFormat = errors.New("incompatible format")

type Store struct {
	Builds  map[string]*Build
	Version uint64
}

type Build struct {
	HighestRecordedBuildID int64
	UnrecordedBuildIDs     map[int64]time.Time
	UpdatedAt              time.Time
}

func (b *Build) SetHighestRecordedBuildID(v int64) {
	b.HighestRecordedBuildID = v
	b.UpdatedAt = time.Now()
}

func (b *Build) UnrecordedBuildIDsIsEmpty() bool {
	return len(b.UnrecordedBuildIDs) == 0
}

func (b *Build) IsUnrecorded(id int64) bool {
	_, exists := b.UnrecordedBuildIDs[id]
	return exists
}

func (b *Build) DeleteUnrecorded(id int64) {
	delete(b.UnrecordedBuildIDs, id)
	b.UpdatedAt = time.Now()
}

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

func New() *Store {
	return &Store{
		Builds:  map[string]*Build{},
		Version: version,
	}
}

func (s *Store) NewBuild(key string, highestRecordedBuildID int64) *Build {
	b := newBuild(highestRecordedBuildID)
	s.Builds[key] = b
	return b
}

func (s *Store) Get(key string) *Build {
	return s.Builds[key]
}

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
