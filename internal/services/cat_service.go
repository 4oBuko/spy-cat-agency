package services

import (
	"context"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
)

type CatService interface {
	AddNewCat(ctx context.Context, cat models.Cat) (models.Cat, error)
	GetCatById(ctx context.Context, id int64) (models.Cat, error)
	UpdateCat(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error)
	DeleteById(ctx context.Context, id int64) error
	GetAllCats(ctx context.Context) ([]models.Cat, error)
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

func (d *DefaultCatService) AddNewCat(ctx context.Context, cat models.Cat) (models.Cat, error) {
	_, err := d.catAPI.GetBreedById(ctx, cat.Breed)
	if err != nil {
		return models.Cat{}, err
	}
	newCat, err := d.catRepo.Add(cat)
	if err != nil {
		return models.Cat{}, err
	}
	return newCat, nil
}

func (d *DefaultCatService) GetCatById(ctx context.Context, id int64) (models.Cat, error) {
	cat, err := d.catRepo.GetById(id)
	if err != nil {
		return models.Cat{}, err
	}
	return cat, nil
}

func (d *DefaultCatService) UpdateCat(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error) {
	err := d.catRepo.Update(id, update)
	if err != nil {
		return models.Cat{}, err
	}
	updatedCat, err := d.catRepo.GetById(id)
	if err != nil {
		return models.Cat{}, err
	}
	return updatedCat, nil
}

func (d *DefaultCatService) DeleteById(ctx context.Context, id int64) error {
	err := d.catRepo.DeleteById(id)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultCatService) GetAllCats(ctx context.Context) ([]models.Cat, error) {
	cats, err := d.catRepo.GetAll()
	if err != nil {
		return nil, err
	}
	return cats, nil
}
