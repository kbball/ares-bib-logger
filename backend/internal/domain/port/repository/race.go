package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type RaceRepository interface {
	List(ctx context.Context, eventID int) ([]entity.Race, error)
	Get(ctx context.Context, id int) (entity.Race, error)
	Create(ctx context.Context, eventID int, name string) (entity.Race, error)
	LockRoster(ctx context.Context, id, rosterCount int) error
	LockOrder(ctx context.Context, id int) error
	SetWinlinkFooterRows(ctx context.Context, id, rows int) error
	// MarkWinlinkExported records that this race's Winlink column was just
	// generated and clears any dismissed reminder, since a fresh export is
	// itself evidence an update was sent.
	MarkWinlinkExported(ctx context.Context, id int) error
	// DismissWinlinkReminder hides the pending reminder until the next export.
	DismissWinlinkReminder(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}
