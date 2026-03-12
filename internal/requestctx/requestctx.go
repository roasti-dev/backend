package requestctx

import "context"

type requestIdKey struct{}
type userIdKey struct{}

var (
	requestIDKey = requestIdKey{}
	userIDKey    = userIdKey{}
)

// GetRequestID returns the requestID from the context, if available
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserID returns the userID from the context, if available
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// WithRequestID puts requestID in context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithRequestID puts userID in context
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}
