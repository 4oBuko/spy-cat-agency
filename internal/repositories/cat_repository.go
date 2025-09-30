package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

var ErrCatNotFound = errors.New("cat not found")

type CatRepository interface {
	GetById(ctx context.Context, id int64) (models.Cat, error)
	GetAll(ctx context.Context) ([]models.Cat, error)
	DeleteById(ctx context.Context, d int64) error
	Update(ctx context.Context, id int64, update models.CatUpdate) error
	Add(ctx context.Context, cat models.Cat) (models.Cat, error)
	IsBusy(ctx context.Context, catId int64) (bool, error)
}

type MySQLCatRepository struct {
	db *sql.DB
}

func NewMySQLCatRepository(db *sql.DB) *MySQLCatRepository {
	return &MySQLCatRepository{db: db}
}

func (m *MySQLCatRepository) GetById(ctx context.Context, id int64) (models.Cat, error) {
	var c models.Cat
	getByIdQuery := "SELECT id, cat_name, breed, years_of_experience, salary FROM cats where id = ?"
	err := m.db.QueryRowContext(ctx, getByIdQuery, id).
		Scan(&c.Id, &c.Name, &c.Breed, &c.YearsOfExperience, &c.Salary)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Cat{}, ErrCatNotFound
		}
		return models.Cat{}, fmt.Errorf("failed to get user by id: %w", err)
	}
	return c, nil
}

func (m *MySQLCatRepository) GetAll(ctx context.Context) ([]models.Cat, error) {
	var cats []models.Cat
	getAllQuery := "SELECT id, cat_name, breed, years_of_experience, salary FROM cats ORDER BY id"
	rows, err := m.db.QueryContext(ctx, getAllQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get all cats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		cat := new(models.Cat)
		if err := rows.Scan(&cat.Id, &cat.Name, &cat.Breed, &cat.YearsOfExperience, &cat.Salary); err != nil {
			return nil, fmt.Errorf("scan failed :%w", err)
		}
		cats = append(cats, *cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return cats, nil
}

func (m *MySQLCatRepository) DeleteById(ctx context.Context, id int64) error {
	exists, err := m.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCatNotFound
	}

	deleteCatQuery := "DELETE FROM cats where id = ?"
	_, err = m.db.ExecContext(ctx, deleteCatQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete cat: %w", err)
	}
	return nil
}

func (m *MySQLCatRepository) Update(ctx context.Context, id int64, update models.CatUpdate) error {
	exists, err := m.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCatNotFound
	}

	updateCatQuery := "UPDATE cats SET salary = ? where id = ?"
	_, err = m.db.ExecContext(ctx, updateCatQuery, update.Salary, id)
	if err != nil {
		return fmt.Errorf("failed to update cat: %w", err)
	}
	return nil
}

func (m *MySQLCatRepository) Add(ctx context.Context, cat models.Cat) (models.Cat, error) {
	newCatQuery := `INSERT INTO cats(cat_name, years_of_experience, salary, breed) VALUES(?,?,?,?)`
	result, err := m.db.ExecContext(ctx, newCatQuery, cat.Name, cat.YearsOfExperience, cat.Salary, cat.Breed)
	if err != nil {
		return models.Cat{}, fmt.Errorf("failed to add new cat: %w", err)
	}

	cat.Id, err = result.LastInsertId()
	if err != nil {
		return models.Cat{}, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return cat, nil
}

func (m *MySQLCatRepository) IsBusy(ctx context.Context, id int64) (bool, error) {
	var busy bool
	isBusyRequest := "SELECT EXISTS (SELECT id, cat_id FROM missions where cat_id = ? and completed = false)"
	err := m.db.QueryRowContext(ctx, isBusyRequest, id).Scan(&busy)
	if err != nil {
		return false, fmt.Errorf("failed to do busy check: %w", err)
	}

	return busy, nil
}

func (m *MySQLCatRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	catExistsQuery := "SELECT EXISTS (SELECT 1 FROM cats WHERE id = ?)"
	err := m.db.QueryRowContext(ctx, catExistsQuery, id).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("existence check failed: %w", err)
	}
	return exists, nil
}
