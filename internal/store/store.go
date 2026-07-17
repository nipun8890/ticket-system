package store

import (
	"errors"
	"sync"
	"time"

	"ticketsystem/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("already exists")
	ErrForbidden = errors.New("forbidden")
)

// Store is a simple in-memory, thread-safe data store. The assignment scope
// explicitly permits in-memory storage, which keeps the service
// dependency-free and trivial to build and run anywhere.
type Store struct {
	mu      sync.RWMutex
	users   map[string]*models.User // keyed by user ID
	byEmail map[string]string       // email -> user ID
	tickets map[string]*models.Ticket
}

func New() *Store {
	return &Store{
		users:   make(map[string]*models.User),
		byEmail: make(map[string]string),
		tickets: make(map[string]*models.Ticket),
	}
}

// CreateUser stores a new user, returning ErrDuplicate if the email is taken.
func (s *Store) CreateUser(email, passwordHash, salt string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[email]; exists {
		return nil, ErrDuplicate
	}

	u := &models.User{
		ID:           newID("usr"),
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		CreatedAt:    time.Now(),
	}
	s.users[u.ID] = u
	s.byEmail[email] = u.ID
	return u, nil
}

// GetUserByEmail returns the user with the given email, or ErrNotFound.
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return s.users[id], nil
}

// CreateTicket stores a new ticket owned by ownerID.
func (s *Store) CreateTicket(ownerID, title, description string) *models.Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	t := &models.Ticket{
		ID:          newID("tkt"),
		OwnerID:     ownerID,
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tickets[t.ID] = t
	return t
}

// ListTicketsByOwner returns all tickets belonging to ownerID.
func (s *Store) ListTicketsByOwner(ownerID string) []*models.Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Ticket, 0)
	for _, t := range s.tickets {
		if t.OwnerID == ownerID {
			result = append(result, t)
		}
	}
	return result
}

// GetTicketForOwner returns the ticket if it exists and belongs to ownerID.
// It returns ErrNotFound if the ticket doesn't exist, and ErrForbidden if it
// exists but belongs to a different user (ownership is never leaked to the
// caller as a 404 vs 403 distinction is intentional here for clarity).
func (s *Store) GetTicketForOwner(ticketID, ownerID string) (*models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tickets[ticketID]
	if !ok {
		return nil, ErrNotFound
	}
	if t.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	return t, nil
}

// UpdateTicketStatus applies a new status to a ticket owned by ownerID.
func (s *Store) UpdateTicketStatus(ticketID, ownerID string, status models.TicketStatus) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[ticketID]
	if !ok {
		return nil, ErrNotFound
	}
	if t.OwnerID != ownerID {
		return nil, ErrForbidden
	}

	t.Status = status
	t.UpdatedAt = time.Now()
	return t, nil
}
