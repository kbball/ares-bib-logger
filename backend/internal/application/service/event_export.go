package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type EventExportImportService struct {
	events      portrepo.EventRepository
	races       portrepo.RaceRepository
	checkpoints portrepo.CheckpointRepository
	runners     portrepo.RunnerRepository
}

func NewEventExportImportService(
	events portrepo.EventRepository,
	races portrepo.RaceRepository,
	checkpoints portrepo.CheckpointRepository,
	runners portrepo.RunnerRepository,
) *EventExportImportService {
	return &EventExportImportService{
		events:      events,
		races:       races,
		checkpoints: checkpoints,
		runners:     runners,
	}
}

var _ portsvc.EventExportService = (*EventExportImportService)(nil)

func (s *EventExportImportService) Export(ctx context.Context, eventID int) (portsvc.EventExportPayload, error) {
	ev, err := s.events.Get(ctx, eventID)
	if err != nil {
		return portsvc.EventExportPayload{}, fmt.Errorf("getting event: %w", err)
	}

	races, err := s.races.List(ctx, eventID)
	if err != nil {
		return portsvc.EventExportPayload{}, fmt.Errorf("listing races: %w", err)
	}

	raceData := make([]portsvc.RaceExportData, 0, len(races))
	for _, race := range races {
		cps, err := s.checkpoints.List(ctx, race.ID)
		if err != nil {
			return portsvc.EventExportPayload{}, fmt.Errorf("listing checkpoints for race %d: %w", race.ID, err)
		}
		sort.Slice(cps, func(i, j int) bool { return cps[i].DisplayOrder < cps[j].DisplayOrder })

		cpExports := make([]portsvc.CheckpointExport, len(cps))
		for i, cp := range cps {
			cpExports[i] = portsvc.CheckpointExport{
				Code:              cp.Code,
				DisplayName:       cp.DisplayName,
				DisplayOrder:      cp.DisplayOrder,
				DistanceFromStart: cp.DistanceFromStart,
			}
		}

		runners, err := s.runners.List(ctx, race.ID)
		if err != nil {
			return portsvc.EventExportPayload{}, fmt.Errorf("listing runners for race %d: %w", race.ID, err)
		}
		sort.Slice(runners, func(i, j int) bool { return runners[i].SortOrder < runners[j].SortOrder })

		runnerExports := make([]portsvc.RunnerExport, len(runners))
		for i, r := range runners {
			runnerExports[i] = portsvc.RunnerExport{
				BibNumber: r.BibNumber,
				FirstName: r.FirstName,
				LastName:  r.LastName,
				SortOrder: r.SortOrder,
			}
		}

		raceData = append(raceData, portsvc.RaceExportData{
			Name:        race.Name,
			Checkpoints: cpExports,
			Runners:     runnerExports,
		})
	}

	return portsvc.EventExportPayload{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Event:      portsvc.EventExportInfo{Name: ev.Name},
		Races:      raceData,
	}, nil
}

func (s *EventExportImportService) Import(ctx context.Context, payload portsvc.EventExportPayload) (int, error) {
	ev, err := s.events.Create(ctx, payload.Event.Name)
	if err != nil {
		return 0, fmt.Errorf("creating event: %w", err)
	}

	for _, raceData := range payload.Races {
		race, err := s.races.Create(ctx, ev.ID, raceData.Name)
		if err != nil {
			return 0, fmt.Errorf("creating race %q: %w", raceData.Name, err)
		}

		for _, cpData := range raceData.Checkpoints {
			_, err := s.checkpoints.Create(ctx, entity.Checkpoint{
				RaceID:            race.ID,
				Code:              cpData.Code,
				DisplayName:       cpData.DisplayName,
				DisplayOrder:      cpData.DisplayOrder,
				DistanceFromStart: cpData.DistanceFromStart,
			})
			if err != nil {
				return 0, fmt.Errorf("creating checkpoint %q: %w", cpData.Code, err)
			}
		}

		if len(raceData.Runners) > 0 {
			runners := make([]entity.Runner, len(raceData.Runners))
			for i, r := range raceData.Runners {
				runners[i] = entity.Runner{
					RaceID:    race.ID,
					BibNumber: r.BibNumber,
					FirstName: r.FirstName,
					LastName:  r.LastName,
					SortOrder: r.SortOrder,
					Status:    entity.StatusUnknown,
				}
			}
			if err := s.runners.BulkCreate(ctx, runners); err != nil {
				return 0, fmt.Errorf("importing runners for race %q: %w", raceData.Name, err)
			}
		}
	}

	return ev.ID, nil
}
