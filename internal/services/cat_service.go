package services

import (
	"context"
	"errors"

	"github.com/4oBuko/spy-cat-agency/internal/models"
	"github.com/4oBuko/spy-cat-agency/internal/myerrors"
	"github.com/4oBuko/spy-cat-agency/internal/repositories"
	"github.com/4oBuko/spy-cat-agency/pkg/catapi"
)

var MaxCatsPerPage = 50
var DefaultCatsPageSize = 10

type CatService interface {
	Add(ctx context.Context, cat models.Cat) (models.Cat, error)
	GetById(ctx context.Context, id int64) (models.Cat, error)
	Update(ctx context.Context, id int64, update models.CatUpdate) (models.Cat, error)
	DeleteById(ctx context.Context, id int64) error
	GetAll(ctx context.Context, query models.PaginationQuery) (models.PaginatedCats, error)
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

func (d *DefaultCatService) GetAll(ctx context.Context, query models.PaginationQuery) (models.PaginatedCats, error) {
	count, err := d.catRepo.GetCount(ctx)
	if err != nil {
		return models.PaginatedCats{}, myerrors.NewServerError(err.Error())
	}
	if query.Size > MaxCatsPerPage {
		return models.PaginatedCats{}, myerrors.NewBadRequestError("page size must be between 0 and 50")
	}

	var offset, limit int
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Size == 0 {
		query.Size = DefaultCatsPageSize
	}

	offset = (query.Page - 1) * query.Size
	limit = query.Size
	totalPages := (count + query.Size - 1) / query.Size
	if query.Page > totalPages {
		return models.PaginatedCats{}, myerrors.NewBadRequestError("request page is greater than total pages")
	}
	cats, err := d.catRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return models.PaginatedCats{}, myerrors.NewServerError(err.Error())
	}

	pCats := models.PaginatedCats{
		Cats: cats,
		Meta: models.Pagination{
			PageSize:   query.Size,
			Page:       query.Page,
			TotalPages: totalPages,
			Total:      count,
		},
	}
	return pCats, nil
}
