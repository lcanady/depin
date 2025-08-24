package main

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
)

// mockProviderRepository is a temporary in-memory repository for development
// This will be replaced with a real database implementation
type mockProviderRepository struct {
	mu        sync.RWMutex
	providers map[uuid.UUID]*types.Provider
	emails    map[string]uuid.UUID // Email to ID mapping for uniqueness
}

func newMockProviderRepository() *mockProviderRepository {
	return &mockProviderRepository{
		providers: make(map[uuid.UUID]*types.Provider),
		emails:    make(map[string]uuid.UUID),
	}
}

// GetByID retrieves a provider by ID
func (r *mockProviderRepository) GetByID(id uuid.UUID) (*types.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[id]
	if !exists {
		return nil, errors.New("provider not found")
	}

	// Return a copy to prevent external modifications
	providerCopy := *provider
	return &providerCopy, nil
}

// GetByEmail retrieves a provider by email
func (r *mockProviderRepository) GetByEmail(email string) (*types.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.emails[email]
	if !exists {
		return nil, errors.New("provider not found")
	}

	provider := r.providers[id]
	// Return a copy to prevent external modifications
	providerCopy := *provider
	return &providerCopy, nil
}

// Create creates a new provider
func (r *mockProviderRepository) Create(provider *types.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if email already exists
	if _, exists := r.emails[provider.Email]; exists {
		return errors.New("provider with this email already exists")
	}

	// Check if ID already exists
	if _, exists := r.providers[provider.ID]; exists {
		return errors.New("provider with this ID already exists")
	}

	// Create a copy to store
	providerCopy := *provider
	r.providers[provider.ID] = &providerCopy
	r.emails[provider.Email] = provider.ID

	return nil
}

// Update updates an existing provider
func (r *mockProviderRepository) Update(provider *types.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if provider exists
	existingProvider, exists := r.providers[provider.ID]
	if !exists {
		return errors.New("provider not found")
	}

	// Check if email change would cause conflict
	if existingProvider.Email != provider.Email {
		if existingID, emailExists := r.emails[provider.Email]; emailExists && existingID != provider.ID {
			return errors.New("provider with this email already exists")
		}

		// Update email mapping
		delete(r.emails, existingProvider.Email)
		r.emails[provider.Email] = provider.ID
	}

	// Update provider
	providerCopy := *provider
	providerCopy.UpdatedAt = time.Now().UTC()
	r.providers[provider.ID] = &providerCopy

	return nil
}

// UpdateLastSeen updates the last seen timestamp for a provider
func (r *mockProviderRepository) UpdateLastSeen(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[id]
	if !exists {
		return errors.New("provider not found")
	}

	// Update last seen time
	now := time.Now().UTC()
	provider.LastSeen = &now

	return nil
}

// ListProviders lists all providers (for admin/debugging purposes)
func (r *mockProviderRepository) ListProviders() ([]*types.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*types.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providerCopy := *provider
		providers = append(providers, &providerCopy)
	}

	return providers, nil
}

// Delete removes a provider (for admin purposes)
func (r *mockProviderRepository) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[id]
	if !exists {
		return errors.New("provider not found")
	}

	delete(r.providers, id)
	delete(r.emails, provider.Email)

	return nil
}

// GetProviderStats returns statistics about providers
func (r *mockProviderRepository) GetProviderStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]int)
	stats["total"] = len(r.providers)

	// Count by status
	for _, provider := range r.providers {
		key := "status_" + string(provider.Status)
		stats[key]++
	}

	return stats
}