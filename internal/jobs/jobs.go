package jobs

// Job type constants — used as Asynq task type names.
const (
	TypePublishPost    = "post:publish"
	TypeRefreshToken   = "social:refresh_token"
	TypeSendEmail      = "email:send"
	TypeStripeWebhook  = "stripe:webhook"
)
