package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type RaceService interface {
	List(ctx context.Context, eventID int) ([]entity.Race, error)
	Get(ctx context.Context, id int) (entity.Race, error)
	Create(ctx context.Context, eventID int, name string) (entity.Race, error)
	Delete(ctx context.Context, id int) error
	LockOrder(ctx context.Context, id int) error
}
