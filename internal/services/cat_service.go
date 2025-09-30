package services

import (
	"context"
	"errors"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/myerrors"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
)

type CatService interface {
	Add(ctx context.Context, cat models.Cat) (models.Cat, error)
	GetById(ctx context.Context, id int64) (models.Cat, error)
	Update(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error)
	DeleteById(ctx context.Context, id int64) error
	GetAll(ctx context.Context) ([]models.Cat, error)
}

type DefaultCatService struct {
	catRepo repositories.CatRepository
	catAPI  catapi.CatAPI
}

func NewDefaultCatService(catRepo repositories.CatRepository, catAPI catapi.CatAPI) *DefaultCatService {
	return &DefaultCatService{
		catRepo: catRepo,
		catAPI:  catAPI,
	}
}

func (d *DefaultCatService) Add(ctx context.Context, cat models.Cat) (models.Cat, error) {
	breed, err := d.catAPI.GetBreedById(ctx, cat.Breed)
	if err != nil {
		if errors.Is(err, catapi.ErrBreedNotFound) {
			return models.Cat{}, myerrors.NewBadRequestError(err.Error())
		}
		return models.Cat{}, myerrors.NewServerError(err.Error())
	}
	cat.Breed = breed.Id
	newCat, err := d.catRepo.Add(ctx, cat)
	if err != nil {
		return models.Cat{}, myerrors.NewServerError(err.Error())
	}
	return newCat, nil
}

func (d *DefaultCatService) GetById(ctx context.Context, id int64) (models.Cat, error) {
	cat, err := d.catRepo.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrCatNotFound) {
			return models.Cat{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Cat{}, myerrors.NewServerError(err.Error())
	}
	return cat, nil
}

func (d *DefaultCatService) Update(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error) {
	err := d.catRepo.Update(ctx, id, update)
	if err != nil {
		if errors.Is(err, repositories.ErrCatNotFound) {
			return models.Cat{}, myerrors.NewNotFoundError(err.Error())
		}
		return models.Cat{}, myerrors.NewServerError(err.Error())
	}
	return d.GetById(ctx, id)
}

func (d *DefaultCatService) DeleteById(ctx context.Context, id int64) error {
	busy, err := d.catRepo.IsBusy(ctx, id)
	if err != nil {
		return myerrors.NewServerError(err.Error())
	}
	if busy {
		return myerrors.NewBadRequestError("cat is busy with a mission. Complete mission before deleting the cat")
	}
	err = d.catRepo.DeleteById(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrCatNotFound) {
			return myerrors.NewNotFoundError(err.Error())
		}
		return myerrors.NewServerError(err.Error())
	}
	return nil
}

func (d *DefaultCatService) GetAll(ctx context.Context) ([]models.Cat, error) {
	cats, err := d.catRepo.GetAll(ctx)
	if err != nil {
		return nil, myerrors.NewServerError(err.Error())
	}
	return cats, nil
}
