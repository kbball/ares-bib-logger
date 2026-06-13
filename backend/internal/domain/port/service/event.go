package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type EventService interface {
	List(ctx context.Context) ([]entity.Event, error)
	Get(ctx context.Context, id int) (entity.Event, error)
	Create(ctx context.Context, name string) (entity.Event, error)
}
