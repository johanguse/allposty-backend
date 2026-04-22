package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// PublishPostPayload is the data serialized into the Asynq task.
type PublishPostPayload struct {
	PostID uuid.UUID `json:"post_id"`
}

// NewPublishPostTask creates an Asynq task for publishing a post.
// opts: asynq.ProcessAt(scheduledAt) to schedule for later.
func NewPublishPostTask(postID uuid.UUID, opts ...asynq.Option) (*asynq.Task, error) {
	payload, err := json.Marshal(PublishPostPayload{PostID: postID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePublishPost, payload, opts...), nil
}

// PublishPostHandler handles TypePublishPost tasks.
// It loads the post, resolves the provider, and calls Publish.
// The actual handler implementation wires in the services/repository layer.
type PublishPostHandler struct {
	// Injected dependencies (set up in cmd/worker/main.go)
	handleFn func(ctx context.Context, postID uuid.UUID) error
}

func NewPublishPostHandler(handleFn func(ctx context.Context, postID uuid.UUID) error) *PublishPostHandler {
	return &PublishPostHandler{handleFn: handleFn}
}

func (h *PublishPostHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p PublishPostPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("publish handler: unmarshal payload: %w", err)
	}
	return h.handleFn(ctx, p.PostID)
}
