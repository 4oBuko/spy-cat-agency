package repositories

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type CatRepository interface {
	GetById(ctx context.Context, id int64) (models.Cat, error)
	GetAll(ctx context.Context) ([]models.Cat, error)
	DeleteById(ctx context.Context, d int64) error
	Update(ctx context.Context, id int64, update models.CatUpdate) error
	Add(ctx context.Context, cat models.Cat) (models.Cat, error)
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
		return models.Cat{}, err
	}
	return c, nil
}

func (m *MySQLCatRepository) GetAll(ctx context.Context) ([]models.Cat, error) {
	var cats []models.Cat
	getAllQuery := "SELECT id, cat_name, breed, years_of_experience, salary FROM cats ORDER BY id"
	rows, err := m.db.QueryContext(ctx, getAllQuery)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		cat := new(models.Cat)
		if err := rows.Scan(&cat.Id, &cat.Name, &cat.Breed, &cat.YearsOfExperience, &cat.Salary); err != nil {
			return nil, err
		}
		cats = append(cats, *cat)
	}
	return cats, nil
}

func (m *MySQLCatRepository) DeleteById(ctx context.Context, id int64) error {
	deleteCatQuery := "DELETE FROM cats where id = ?"
	res, err := m.db.ExecContext(ctx, deleteCatQuery, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *MySQLCatRepository) Update(ctx context.Context, id int64, update models.CatUpdate) error {
	updateCatQuery := "UPDATE cats SET salary = ? where id = ?"
	res, err := m.db.ExecContext(ctx, updateCatQuery, update.Salary, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *MySQLCatRepository) Add(ctx context.Context, cat models.Cat) (models.Cat, error) {
	newCatQuery := `INSERT INTO cats(cat_name, years_of_experience, salary, breed) VALUES(?,?,?,?)`
	result, err := m.db.ExecContext(ctx, newCatQuery, cat.Name, cat.YearsOfExperience, cat.Salary, cat.Breed)
	if err != nil {
		return models.Cat{}, err
	}

	cat.Id, err = result.LastInsertId()
	if err != nil {
		return models.Cat{}, err
	}
	return cat, nil
}
