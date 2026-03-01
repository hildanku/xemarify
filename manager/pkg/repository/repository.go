package repository

import "context"

type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	FindByUUID(ctx context.Context, uuid string) (*T, error)
	Delete(ctx context.Context, uuid string) (*T, error)
}
