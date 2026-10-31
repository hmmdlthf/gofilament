package activitylog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrActivityNotFound = errors.New("activity log not found")
)

// Filter specifies criteria for querying activity records.
type Filter struct {
	LogName     string
	SubjectType string
	SubjectID   string
	CauserType  string
	CauserID    string
	Event       string
	Search      string
	Page        int
	PerPage     int
}

// Store defines the storage and querying contract for audit activity logs.
type Store interface {
	Log(ctx context.Context, activity *Activity) error
	GetByID(ctx context.Context, id string) (*Activity, error)
	GetBySubject(ctx context.Context, subjectType, subjectID string, limit ...int) ([]*Activity, error)
	GetByCauser(ctx context.Context, causerType, causerID string, limit ...int) ([]*Activity, error)
	Query(ctx context.Context, filter Filter) ([]*Activity, int, error)
	Clear(ctx context.Context) error
}

// MemoryStore provides a thread-safe, in-memory implementation of Store.
type MemoryStore struct {
	mu         sync.RWMutex
	activities []*Activity
	seq        uint64
}

// NewMemoryStore creates an empty in-memory activity log store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		activities: make([]*Activity, 0),
	}
}

// Log records a new activity in memory.
func (s *MemoryStore) Log(ctx context.Context, activity *Activity) error {
	if activity == nil {
		return errors.New("cannot log nil activity")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if activity.ID == "" {
		idNum := atomic.AddUint64(&s.seq, 1)
		activity.ID = fmt.Sprintf("act_%d_%d", time.Now().UnixNano(), idNum)
	}
	if activity.CreatedAt.IsZero() {
		activity.CreatedAt = time.Now()
	}

	// Prepend newest first
	s.activities = append([]*Activity{activity}, s.activities...)
	return nil
}

// GetByID finds an activity by unique identifier.
func (s *MemoryStore) GetByID(ctx context.Context, id string) (*Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.activities {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, ErrActivityNotFound
}

// GetBySubject retrieves activities targeting a specific subject record.
func (s *MemoryStore) GetBySubject(ctx context.Context, subjectType, subjectID string, limit ...int) ([]*Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	maxItems := 50
	if len(limit) > 0 && limit[0] > 0 {
		maxItems = limit[0]
	}

	res := make([]*Activity, 0)
	for _, a := range s.activities {
		if strings.EqualFold(a.SubjectType, subjectType) && a.SubjectID == subjectID {
			res = append(res, a)
			if len(res) >= maxItems {
				break
			}
		}
	}
	return res, nil
}

// GetByCauser retrieves activities initiated by a specific causer/actor.
func (s *MemoryStore) GetByCauser(ctx context.Context, causerType, causerID string, limit ...int) ([]*Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	maxItems := 50
	if len(limit) > 0 && limit[0] > 0 {
		maxItems = limit[0]
	}

	res := make([]*Activity, 0)
	for _, a := range s.activities {
		if strings.EqualFold(a.CauserType, causerType) && a.CauserID == causerID {
			res = append(res, a)
			if len(res) >= maxItems {
				break
			}
		}
	}
	return res, nil
}

// Query filters and paginates activities according to criteria.
func (s *MemoryStore) Query(ctx context.Context, filter Filter) ([]*Activity, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]*Activity, 0)
	searchLower := strings.ToLower(strings.TrimSpace(filter.Search))

	for _, a := range s.activities {
		if filter.LogName != "" && !strings.EqualFold(a.LogName, filter.LogName) {
			continue
		}
		if filter.SubjectType != "" && !strings.EqualFold(a.SubjectType, filter.SubjectType) {
			continue
		}
		if filter.SubjectID != "" && a.SubjectID != filter.SubjectID {
			continue
		}
		if filter.CauserType != "" && !strings.EqualFold(a.CauserType, filter.CauserType) {
			continue
		}
		if filter.CauserID != "" && a.CauserID != filter.CauserID {
			continue
		}
		if filter.Event != "" && !strings.EqualFold(a.Event, filter.Event) {
			continue
		}

		if searchLower != "" {
			descMatch := strings.Contains(strings.ToLower(a.Description), searchLower)
			causerMatch := strings.Contains(strings.ToLower(a.CauserName), searchLower)
			subjectMatch := strings.Contains(strings.ToLower(a.SubjectType), searchLower) || strings.Contains(strings.ToLower(a.SubjectID), searchLower)
			ipMatch := strings.Contains(strings.ToLower(a.IPAddress), searchLower)
			eventMatch := strings.Contains(strings.ToLower(a.Event), searchLower)
			if !descMatch && !causerMatch && !subjectMatch && !ipMatch && !eventMatch {
				continue
			}
		}

		matched = append(matched, a)
	}

	total := len(matched)
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage <= 0 {
		perPage = 25
	}

	start := (page - 1) * perPage
	if start >= total {
		return []*Activity{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

// Clear removes all activities from the memory store.
func (s *MemoryStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activities = make([]*Activity, 0)
	return nil
}

var (
	defaultStoreMu sync.RWMutex
	defaultStore   Store = NewMemoryStore()
)

// SetDefaultStore registers the global default store for activity logging.
func SetDefaultStore(s Store) {
	defaultStoreMu.Lock()
	defer defaultStoreMu.Unlock()
	defaultStore = s
}

// GetDefaultStore returns the global default store instance.
func GetDefaultStore() Store {
	defaultStoreMu.RLock()
	defer defaultStoreMu.RUnlock()
	return defaultStore
}
