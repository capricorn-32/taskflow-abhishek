package httpapi

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const (
	userIDKey ctxKey = "user_id"
	emailKey  ctxKey = "email"
)

func WithUser(ctx context.Context, userID uuid.UUID, email string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	return ctx
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

func EmailFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(emailKey)
	email, ok := v.(string)
	return email, ok
}
