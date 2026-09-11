package accessticket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Spec struct {
	User      string
	SessionID string
	Method    string
	Route     string
	Resource  string
	Once      bool
	TTL       time.Duration
}

type Ticket struct {
	User      string
	SessionID string
	Method    string
	Route     string
	Resource  string
	Once      bool
	Expiry    time.Time
}

type SessionChecker func(sessionID string) bool

type Store struct {
	mu        sync.Mutex
	tickets   map[string]Ticket
	sessionOK SessionChecker
}

func NewStore() *Store {
	return &Store{tickets: make(map[string]Ticket)}
}

var Default = NewStore()

func (s *Store) SetSessionChecker(fn SessionChecker) {
	s.mu.Lock()
	s.sessionOK = fn
	s.mu.Unlock()
}

func (s *Store) Issue(spec Spec) (string, error) {
	if spec.User == "" || spec.SessionID == "" || spec.Method == "" || spec.Route == "" {
		return "", fmt.Errorf("incomplete ticket spec")
	}
	if spec.TTL <= 0 {
		spec.TTL = time.Minute
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw[:])
	s.mu.Lock()
	s.pruneLocked(time.Now())
	s.tickets[id] = Ticket{
		User:      spec.User,
		SessionID: spec.SessionID,
		Method:    spec.Method,
		Route:     spec.Route,
		Resource:  spec.Resource,
		Once:      spec.Once,
		Expiry:    time.Now().Add(spec.TTL),
	}
	s.mu.Unlock()
	return id, nil
}

func (s *Store) Authorize(id, method, route, resource string) (Ticket, error) {
	s.mu.Lock()
	now := time.Now()
	s.pruneLocked(now)
	ticket, ok := s.tickets[id]
	if !ok || now.After(ticket.Expiry) {
		delete(s.tickets, id)
		s.mu.Unlock()
		return Ticket{}, fmt.Errorf("invalid ticket")
	}
	if ticket.Method != method || ticket.Route != route {
		s.mu.Unlock()
		return Ticket{}, fmt.Errorf("invalid ticket")
	}
	if ticket.Resource != "" && ticket.Resource != resource {
		s.mu.Unlock()
		return Ticket{}, fmt.Errorf("invalid ticket")
	}
	checker := s.sessionOK
	sessionID := ticket.SessionID
	once := ticket.Once
	s.mu.Unlock()

	if checker != nil && !checker(sessionID) {
		s.mu.Lock()
		delete(s.tickets, id)
		s.mu.Unlock()
		return Ticket{}, fmt.Errorf("invalid ticket")
	}
	if once {
		s.mu.Lock()
		current, still := s.tickets[id]
		if !still || current.SessionID != sessionID {
			s.mu.Unlock()
			return Ticket{}, fmt.Errorf("invalid ticket")
		}
		delete(s.tickets, id)
		s.mu.Unlock()
	}
	return ticket, nil
}

func (s *Store) RevokeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, ticket := range s.tickets {
		if ticket.SessionID == sessionID {
			delete(s.tickets, id)
		}
	}
}

func (s *Store) RevokeSessions(ids map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, ticket := range s.tickets {
		if _, ok := ids[ticket.SessionID]; ok {
			delete(s.tickets, id)
		}
	}
}

func (s *Store) pruneLocked(now time.Time) {
	for id, ticket := range s.tickets {
		if now.After(ticket.Expiry) {
			delete(s.tickets, id)
		}
	}
}

func DownloadSpec(user, sessionID, path string) Spec {
	return Spec{
		User: user, SessionID: sessionID,
		Method: http.MethodGet, Route: "/api/v1/files/download",
		Resource: path, TTL: time.Minute,
	}
}

func TerminalSpec(user, sessionID string) Spec {
	return Spec{
		User: user, SessionID: sessionID,
		Method: http.MethodGet, Route: "/api/v1/terminal",
		Once: true, TTL: time.Minute,
	}
}
