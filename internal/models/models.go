package models

import "time"

// TicketStatus represents the allowed states of a ticket.
type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusClosed     TicketStatus = "closed"
)

// IsValid reports whether s is one of the supported ticket statuses.
func IsValid(s TicketStatus) bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	}
	return false
}

// allowedTransitions defines the only legal forward moves in the status flow:
// open -> in_progress -> closed. A closed ticket can never move again.
var allowedTransitions = map[TicketStatus]map[TicketStatus]bool{
	StatusOpen:       {StatusInProgress: true},
	StatusInProgress: {StatusClosed: true},
	StatusClosed:     {},
}

// CanTransition reports whether moving from `from` to `to` is a legal transition.
func CanTransition(from, to TicketStatus) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return next[to]
}

// User represents a registered account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Ticket represents a support ticket owned by a user.
type Ticket struct {
	ID          string       `json:"id"`
	OwnerID     string       `json:"owner_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
