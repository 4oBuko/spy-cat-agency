package services

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
)

type MissionService interface {
	Add(ctx context.Context, mission models.Mission) (models.Mission, error)
	GetById(ctx context.Context, id int64) (models.Mission, error)
}

type DefaultMissionService struct {
	missionReposity  repositories.TxMissionRepository
	targetRepository repositories.TxTargetRepository
}

func NewDefaultMissionService(missionRepo repositories.TxMissionRepository, targetRepository repositories.TxTargetRepository) *DefaultMissionService {
	return &DefaultMissionService{
		missionReposity:  missionRepo,
		targetRepository: targetRepository,
	}
}

func (d *DefaultMissionService) Add(ctx context.Context, mission models.Mission) (models.Mission, error) {
	savedMission, err := d.missionReposity.WithTransaction(ctx,
		func(tx *sql.Tx) (models.Mission, error) {
			sm, err := d.missionReposity.AddWithTx(ctx, tx, mission)
			sm.Targets = nil // delete unsaved targets
			if err != nil {
				return models.Mission{}, err
			}
			for _, t := range mission.Targets {
				t.MissionId = sm.Id
				nt, err := d.targetRepository.AddWithTx(ctx, tx, t)
				if err != nil {
					return models.Mission{}, err
				}
				sm.Targets = append(sm.Targets, nt)
			}
			return sm, nil
		})
	if err != nil {
		return models.Mission{}, err
	}
	return savedMission, nil
}

func (d *DefaultMissionService) GetById(ctx context.Context, id int64) (models.Mission, error) {
	mission, err := d.missionReposity.GetById(ctx, id)
	if err != nil {
		return models.Mission{}, err
	}
	targets, err := d.targetRepository.GetByMissionId(ctx, mission.Id)
	mission.Targets = targets
	if err != nil {
		return models.Mission{}, err
	}
	return mission, nil
}
