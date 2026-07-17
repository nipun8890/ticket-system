package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ticketsystem/internal/auth"
	"ticketsystem/internal/middleware"
	"ticketsystem/internal/models"
	"ticketsystem/internal/store"
)

const tokenTTL = 24 * time.Hour

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Handler bundles the store and JWT secret needed by all HTTP handlers.
type Handler struct {
	store  *store.Store
	secret string
}

func New(s *store.Store, jwtSecret string) *Handler {
	return &Handler{store: s, secret: jwtSecret}
}

// ---------- shared response helpers ----------

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// ---------- health ----------

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- auth ----------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  publicUser `json:"user"`
}

type publicUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, salt, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user, err := h.store.CreateUser(req.Email, hash, salt)
	if err != nil {
		if err == store.ErrDuplicate {
			writeError(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := auth.GenerateToken(h.secret, user.ID, user.Email, tokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  publicUser{ID: user.ID, Email: user.Email},
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !auth.VerifyPassword(req.Password, user.Salt, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(h.secret, user.ID, user.Email, tokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  publicUser{ID: user.ID, Email: user.Email},
	})
}

// ---------- tickets ----------

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func userIDFromContext(r *http.Request) string {
	if v, ok := r.Context().Value(middleware.UserIDKey).(string); ok {
		return v
	}
	return ""
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	ownerID := userIDFromContext(r)

	var req createTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	ticket := h.store.CreateTicket(ownerID, req.Title, strings.TrimSpace(req.Description))
	writeJSON(w, http.StatusCreated, ticket)
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	ownerID := userIDFromContext(r)
	tickets := h.store.ListTicketsByOwner(ownerID)
	writeJSON(w, http.StatusOK, map[string]any{"tickets": tickets})
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	ownerID := userIDFromContext(r)
	ticketID := r.PathValue("id")

	ticket, err := h.store.GetTicketForOwner(ticketID, ownerID)
	if err != nil {
		handleTicketLookupErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	ownerID := userIDFromContext(r)
	ticketID := r.PathValue("id")

	var req updateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newStatus := models.TicketStatus(strings.TrimSpace(strings.ToLower(req.Status)))
	if !models.IsValid(newStatus) {
		writeError(w, http.StatusBadRequest, "status must be one of: open, in_progress, closed")
		return
	}

	current, err := h.store.GetTicketForOwner(ticketID, ownerID)
	if err != nil {
		handleTicketLookupErr(w, err)
		return
	}

	if current.Status == newStatus {
		writeError(w, http.StatusConflict, "ticket is already in this status")
		return
	}

	if !models.CanTransition(current.Status, newStatus) {
		writeError(w, http.StatusConflict, "invalid status transition: "+string(current.Status)+" -> "+string(newStatus))
		return
	}

	updated, err := h.store.UpdateTicketStatus(ticketID, ownerID, newStatus)
	if err != nil {
		handleTicketLookupErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func handleTicketLookupErr(w http.ResponseWriter, err error) {
	switch err {
	case store.ErrNotFound:
		writeError(w, http.StatusNotFound, "ticket not found")
	case store.ErrForbidden:
		// A ticket that exists but belongs to someone else is reported as
		// 404 to avoid leaking the existence of other users' tickets.
		writeError(w, http.StatusNotFound, "ticket not found")
	default:
		writeError(w, http.StatusInternalServerError, "unexpected error")
	}
}
