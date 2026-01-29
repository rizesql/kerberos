package platform

import (
	"sync"

	"github.com/rizesql/kerberos/internal/protocol"
)

// TicketCache stores the TGT and service tickets in memory
type TicketCache struct {
	mu sync.RWMutex

	// TGT (Ticket Granting Ticket) - stored as EncryptedData
	tgt *protocol.EncryptedData

	// Session key from AS-REP (for encrypting TGS authenticator)
	tgtSessionKey *protocol.SessionKey

	// Client principal (alice@ATHENA.MIT.EDU)
	clientPrincipal protocol.Principal

	// Service tickets keyed by service name (e.g., "http/api-server")
	serviceTickets map[string]*protocol.EncryptedData

	// Service session keys keyed by service name
	serviceSessionKeys map[string]*protocol.SessionKey
}

func NewTicketCache() *TicketCache {
	return &TicketCache{
		serviceTickets:     make(map[string]*protocol.EncryptedData),
		serviceSessionKeys: make(map[string]*protocol.SessionKey),
	}
}

// StoreTGT stores the TGT
func (tc *TicketCache) StoreTGT(tgt protocol.EncryptedData) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tgt = &tgt
}

// StoreTGTWithSession stores the TGT along with session key and client principal
func (tc *TicketCache) StoreTGTWithSession(tgt protocol.EncryptedData, sessionKey protocol.SessionKey, client protocol.Principal) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tgt = &tgt
	tc.tgtSessionKey = &sessionKey
	tc.clientPrincipal = client
}

// GetTGT retrieves the stored TGT
func (tc *TicketCache) GetTGT() *protocol.EncryptedData {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.tgt
}

// GetTGTSessionKey retrieves the session key from AS-REP
func (tc *TicketCache) GetTGTSessionKey() *protocol.SessionKey {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.tgtSessionKey
}

// GetClientPrincipal retrieves the client principal
func (tc *TicketCache) GetClientPrincipal() protocol.Principal {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.clientPrincipal
}

// StoreServiceTicket stores a service ticket
func (tc *TicketCache) StoreServiceTicket(service string, ticket protocol.EncryptedData) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.serviceTickets[service] = &ticket
}

// StoreServiceTicketWithSession stores a service ticket with its session key
func (tc *TicketCache) StoreServiceTicketWithSession(service string, ticket protocol.EncryptedData, sessionKey protocol.SessionKey) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.serviceTickets[service] = &ticket
	tc.serviceSessionKeys[service] = &sessionKey
}

// GetServiceTicket retrieves a service ticket
func (tc *TicketCache) GetServiceTicket(service string) *protocol.EncryptedData {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.serviceTickets[service]
}

// GetServiceSessionKey retrieves the session key for a service
func (tc *TicketCache) GetServiceSessionKey(service string) *protocol.SessionKey {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.serviceSessionKeys[service]
}
