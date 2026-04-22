package providers

import (
	"fmt"

	"github.com/allposty/allposty-backend/internal/config"
)

// Registry maps platform names to their initialized providers.
type Registry struct {
	providers map[Platform]SocialProvider
}

func NewRegistry(cfg *config.Config) *Registry {
	r := &Registry{providers: make(map[Platform]SocialProvider)}

	r.register(NewInstagramProvider(cfg))
	r.register(NewFacebookProvider(cfg))
	r.register(NewLinkedInProvider(cfg))
	r.register(NewTwitterProvider(cfg))
	r.register(NewTikTokProvider(cfg))
	r.register(NewYouTubeProvider(cfg))

	return r
}

func (r *Registry) register(p SocialProvider) {
	r.providers[p.Platform()] = p
}

func (r *Registry) Get(platform Platform) (SocialProvider, error) {
	p, ok := r.providers[platform]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", platform)
	}
	return p, nil
}

func (r *Registry) All() map[Platform]SocialProvider {
	return r.providers
}
