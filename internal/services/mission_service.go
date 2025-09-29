package services

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/myerrors"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
)

type MissionService interface {
	Add(ctx context.Context, mission models.Mission) (models.Mission, error)
	GetById(ctx context.Context, id int64) (models.Mission, error)
	GetAll(ctx context.Context) ([]models.Mission, error)
	Assign(ctx context.Context, missionId, catId int64) error
	CompleteTarget(ctx context.Context, missionId, targetId int64) error
	UpdateTarget(ctx context.Context, missionId, targetId int64, update models.TargetUpdate) (models.Target, error)
	DeleteTarget(ctx context.Context, missionId, targetId int64) error
	AddTarget(ctx context.Context, missionId int64, target models.Target) (models.Mission, error)
	Complete(ctx context.Context, missionId int64) (models.Mission, error)
	Delete(ctx context.Context, missionId int64) error
}

type DefaultMissionService struct {
	missionReposity  repositories.TxMissionRepository
	targetRepository repositories.TxTargetRepository
	catRepository    repositories.CatRepository
}

func NewDefaultMissionService(mr repositories.TxMissionRepository, tr repositories.TxTargetRepository, cr repositories.CatRepository) *DefaultMissionService {
	return &DefaultMissionService{
		missionReposity:  mr,
		targetRepository: tr,
		catRepository:    cr,
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

func (d *DefaultMissionService) GetAll(ctx context.Context) ([]models.Mission, error) {
	missions, err := d.missionReposity.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range missions {
		targets, err := d.targetRepository.GetByMissionId(ctx, missions[i].Id)
		if err != nil {
			return nil, err
		}
		missions[i].Targets = targets
	}
	return missions, nil
}

func (d *DefaultMissionService) Assign(ctx context.Context, missionId, catId int64) error {
	mission, err := d.missionReposity.GetById(ctx, missionId)
	if err != nil {
		return err
	}
	if mission.CatId != 0 {
		return &myerrors.RequestError{Message: "mission is already assigned"}
	}
	if mission.Completed {
		return &myerrors.RequestError{Message: "cannot assign cat to a completed mission"}
	}
	_, err = d.catRepository.GetById(ctx, catId)
	if err != nil {
		return err
	}
	busy, err := d.catRepository.IsBusy(ctx, catId)
	if err != nil {
		return err
	}
	if busy {
		return &myerrors.RequestError{Message: "cat is busy with another mission"}
	}
	err = d.missionReposity.Assign(ctx, missionId, catId)
	if err != nil {
		return err
	}

	return nil
}

func (d *DefaultMissionService) CompleteTarget(ctx context.Context, missionId, targetId int64) error {
	mission, err := d.missionReposity.GetById(ctx, missionId)
	if err != nil {
		return err
	}
	if mission.Completed {
		return &myerrors.RequestError{"Mission is already completed"}
	}
	if mission.CatId == 0 {
		return &myerrors.RequestError{"Mission is not assigned to anybody"}
	}
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		return err
	}
	if target.Completed {
		return &myerrors.RequestError{"Target is already completed"}
	}
	err = d.targetRepository.Complete(ctx, targetId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultMissionService) UpdateTarget(ctx context.Context, missionId, targetId int64, update models.TargetUpdate) (models.Target, error) {
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		return models.Target{}, err
	}
	if target.Completed {
		return models.Target{}, &myerrors.RequestError{"Target is already completed"}
	}
	if target.MissionId != missionId {
		return models.Target{}, &myerrors.RequestError{"Target is not related to this mission"}
	}

	err = d.targetRepository.Update(ctx, targetId, update)
	if err != nil {
		return models.Target{}, err
	}

	updatedTarget, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		return models.Target{}, err
	}
	return updatedTarget, nil
}

func (d *DefaultMissionService) DeleteTarget(ctx context.Context, missionId, targetId int64) error {
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		return err
	}
	if target.Completed {
		return &myerrors.RequestError{"Target is already completed"}
	}
	if target.MissionId != missionId {
		return &myerrors.RequestError{"Target is not related to this mission"}
	}
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		return err
	}
	if mission.Completed {
		return &myerrors.RequestError{"Mission is already completed"}
	}
	if len(mission.Targets) == 1 {
		return &myerrors.RequestError{"Mission must have at least one target"}
	}

	err = d.targetRepository.Delete(ctx, targetId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultMissionService) AddTarget(ctx context.Context, missionId int64, target models.Target) (models.Mission, error) {
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		return models.Mission{}, err
	}
	if mission.Completed {
		return models.Mission{}, &myerrors.RequestError{"Mission is already completed"}
	}
	if len(mission.Targets) == 3 {
		return models.Mission{}, &myerrors.RequestError{"Mission cannot have more than 3 targets"}
	}
	target.MissionId = missionId
	nTarget, err := d.targetRepository.Add(ctx, target)
	if err != nil {
		return models.Mission{}, err
	}
	mission.Targets = append(mission.Targets, nTarget)
	return mission, nil
}

func (d *DefaultMissionService) Complete(ctx context.Context, id int64) (models.Mission, error) {
	mission, err := d.GetById(ctx, id)
	if err != nil {
		return models.Mission{}, err
	}
	if mission.CatId == 0 {
		return models.Mission{}, &myerrors.RequestError{"mission must be assigned first"}
	}
	if mission.Completed {
		return models.Mission{}, &myerrors.RequestError{"mission is already completed"}
	}
	for i := range mission.Targets {
		if !mission.Targets[i].Completed {
			return models.Mission{}, &myerrors.RequestError{"mission has uncompleted targets"}
		}
	}
	err = d.missionReposity.Complete(ctx, id)
	if err != nil {
		return models.Mission{}, err
	}
	mission.Completed = true
	return mission, nil
}

func (d *DefaultMissionService) Delete(ctx context.Context, missionId int64) error {
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		return err
	}
	if mission.CatId != 0 {
		return &myerrors.RequestError{"mission is already assigned"}
	}
	err = d.missionReposity.Delete(ctx, missionId)
	if err != nil {
		return err
	}
	return nil
}
