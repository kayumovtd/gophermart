package auth

import "context"

type userIDContextKey struct{}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func UserID(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDContextKey{})
	userID, ok := v.(int64)
	return userID, ok
}
