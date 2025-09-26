package repositories

import (
	"database/sql"
	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type CatRepository interface {
	GetById(id int64) (models.Cat, error)
	GetAll() ([]models.Cat, error)
	DeleteById(id int64) error
	Update(id int64, update models.CatUpdate) error
	Add(cat models.Cat) (models.Cat, error)
}

type MySQLCatRepository struct {
	connection *sql.DB
}

func NewMySQLCatRepo(connection *sql.DB) *MySQLCatRepository {
	return &MySQLCatRepository{connection: connection}
}

func (m *MySQLCatRepository) GetById(id int64) (models.Cat, error) {
	var c models.Cat
	getByIdQuery := "SELECT id, cat_name, breed, years_of_experience, salary FROM cats where id = ?"
	err := m.connection.QueryRow(getByIdQuery, id).
		Scan(&c.Id, &c.Name, &c.Breed, &c.YearsOfExperience, &c.Salary)
	if err != nil {
		return models.Cat{}, err
	}
	return c, nil
}

func (m *MySQLCatRepository) GetAll() ([]models.Cat, error) {
	var cats []models.Cat
	getAllQuery := "SELECT id, cat_name, breed, years_of_experience, salary FROM cats ORDER BY id"
	rows, err := m.connection.Query(getAllQuery)
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

func (m *MySQLCatRepository) DeleteById(id int64) error {
	deleteCatQuery := "DELETE FROM cats where id = ?"
	res, err := m.connection.Exec(deleteCatQuery, id)
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

func (m *MySQLCatRepository) Update(id int64, update models.CatUpdate) error {
	updateCatQuery := "UPDATE cats SET salary = ? where id = ?"
	res, err := m.connection.Exec(updateCatQuery, update.Salary, id)
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

func (m *MySQLCatRepository) Add(cat models.Cat) (models.Cat, error) {
	newCatQuery := `INSERT INTO cats(cat_name, years_of_experience, salary, breed) VALUES(?,?,?,?)`
	result, err := m.connection.Exec(newCatQuery, cat.Name, cat.YearsOfExperience, cat.Salary, cat.Breed)
	if err != nil {
		return models.Cat{}, err
	}

	cat.Id, err = result.LastInsertId()
	return cat, nil
}
