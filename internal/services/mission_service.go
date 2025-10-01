package services

import (
	"context"
	"database/sql"
	"errors"

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
			if err != nil {
				return models.Mission{}, err
			}

			sm.Targets = nil // delete unsaved targets
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
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}

	return savedMission, nil
}

func (d *DefaultMissionService) GetById(ctx context.Context, id int64) (models.Mission, error) {
	mission, err := d.missionReposity.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return models.Mission{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}
	targets, err := d.targetRepository.GetByMissionId(ctx, mission.Id)
	if err != nil {
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}
	mission.Targets = targets

	return mission, nil
}

func (d *DefaultMissionService) GetAll(ctx context.Context) ([]models.Mission, error) {
	missions, err := d.missionReposity.GetAll(ctx)
	if err != nil {
		return nil, myerrors.NewServerError(err.Error())
	}
	for i := range missions {
		targets, err := d.targetRepository.GetByMissionId(ctx, missions[i].Id)
		if err != nil {
			return nil, myerrors.NewServerError(err.Error())
		}
		missions[i].Targets = targets
	}
	return missions, nil
}

func (d *DefaultMissionService) Assign(ctx context.Context, missionId, catId int64) error {
	mission, err := d.missionReposity.GetById(ctx, missionId)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	if mission.CatId != 0 {
		return myerrors.NewBadRequestError("mission is already assigned")
	}
	if mission.Completed {
		return myerrors.NewBadRequestError("cannot assign cat to a completed mission")
	}
	_, err = d.catRepository.GetById(ctx, catId)
	if err != nil {
		if errors.Is(err, repositories.ErrCatNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	busy, err := d.catRepository.IsBusy(ctx, catId)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}
	if busy {
		return myerrors.NewBadRequestError("cat is busy with another mission")
	}
	err = d.missionReposity.Assign(ctx, missionId, catId)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}

	return nil
}

func (d *DefaultMissionService) CompleteTarget(ctx context.Context, missionId, targetId int64) error {
	mission, err := d.missionReposity.GetById(ctx, missionId)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	if mission.Completed {
		return myerrors.NewBadRequestError("mission is already completed")
	}
	if mission.CatId == 0 {
		return myerrors.NewBadRequestError("mission is not assigned to anybody")
	}
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}
	if target.Completed {
		return myerrors.NewBadRequestError("target is already completed")
	}
	err = d.targetRepository.Complete(ctx, targetId)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}
	return nil
}

func (d *DefaultMissionService) UpdateTarget(ctx context.Context, missionId, targetId int64, update models.TargetUpdate) (models.Target, error) {
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		if errors.Is(err, repositories.ErrTargetNotFound) {
			return models.Target{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Target{}, myerrors.NewServerError(err.Error())
	}
	if target.Completed {
		return models.Target{}, myerrors.NewBadRequestError("Target is already completed")
	}
	if target.MissionId != missionId {
		return models.Target{}, myerrors.NewBadRequestError("Target is not related to this mission")
	}

	err = d.targetRepository.Update(ctx, targetId, update)
	if err != nil {
		if errors.Is(err, repositories.ErrTargetNotFound) {
			return models.Target{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Target{}, myerrors.NewServerError(err.Error())

	}

	updatedTarget, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		if errors.Is(err, repositories.ErrTargetNotFound) {
			return models.Target{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Target{}, myerrors.NewServerError(err.Error())
	}
	return updatedTarget, nil
}

func (d *DefaultMissionService) DeleteTarget(ctx context.Context, missionId, targetId int64) error {
	target, err := d.targetRepository.GetById(ctx, targetId)
	if err != nil {
		if errors.Is(err, repositories.ErrTargetNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	if target.Completed {
		return myerrors.NewBadRequestError("Target is already completed")
	}
	if target.MissionId != missionId {
		return myerrors.NewBadRequestError("Target is not related to this mission")
	}
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		return err
	}
	if mission.Completed {
		return myerrors.NewBadRequestError("Mission is already completed")
	}
	if len(mission.Targets) == 1 {
		return myerrors.NewBadRequestError("Mission must have at least one target")
	}

	err = d.targetRepository.Delete(ctx, targetId)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}
	return nil
}

func (d *DefaultMissionService) AddTarget(ctx context.Context, missionId int64, target models.Target) (models.Mission, error) {
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return models.Mission{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}
	if mission.Completed {
		return models.Mission{}, myerrors.NewBadRequestError("Mission is already completed")
	}
	if len(mission.Targets) == 3 {
		return models.Mission{}, myerrors.NewBadRequestError("Mission cannot have more than 3 targets")
	}
	target.MissionId = missionId
	nTarget, err := d.targetRepository.Add(ctx, target)
	if err != nil {
		return models.Mission{}, myerrors.NewServerError(err.Error())

	}
	mission.Targets = append(mission.Targets, nTarget)
	return mission, nil
}

func (d *DefaultMissionService) Complete(ctx context.Context, id int64) (models.Mission, error) {
	mission, err := d.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return models.Mission{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}
	if mission.CatId == 0 {
		return models.Mission{}, myerrors.NewBadRequestError("mission must be assigned first")
	}
	if mission.Completed {
		return models.Mission{}, myerrors.NewBadRequestError("mission is already completed")
	}
	for i := range mission.Targets {
		if !mission.Targets[i].Completed {
			return models.Mission{}, myerrors.NewBadRequestError("mission has uncompleted targets")
		}
	}
	err = d.missionReposity.Complete(ctx, id)
	if err != nil {
		return models.Mission{}, myerrors.NewServerError(err.Error())
	}
	mission.Completed = true
	return mission, nil
}

func (d *DefaultMissionService) Delete(ctx context.Context, missionId int64) error {
	mission, err := d.GetById(ctx, missionId)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	if mission.CatId != 0 {
		return myerrors.NewBadRequestError("mission is already assigned")
	}
	err = d.missionReposity.Delete(ctx, missionId)
	if err != nil {
		if errors.Is(err, repositories.ErrMissionNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	return nil
}
