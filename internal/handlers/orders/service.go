package orders

import (
	"context"

	coreorders "github.com/kayumovtd/gophermart/internal/orders"
)

type Service interface {
	Upload(ctx context.Context, userID int64, number string) (coreorders.UploadResult, error)
	List(ctx context.Context, userID int64) ([]coreorders.Order, error)
}
