package repositories

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type MissionRepository interface {
	Add(ctx context.Context, mission models.Mission) (models.Mission, error)
	GetById(ctx context.Context, id int64) (models.Mission, error)
	GetAll(ctx context.Context) ([]models.Mission, error)
	Assign(ctx context.Context, missionId, catId int64) error
	Complete(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}

type TxMissionRepository interface {
	MissionRepository
	AddWithTx(ctx context.Context, tx *sql.Tx, mission models.Mission) (models.Mission, error)
	WithTransaction(ctx context.Context, fn func(*sql.Tx) (models.Mission, error)) (models.Mission, error)
}

type MySQLMissionRepository struct {
	db *sql.DB
}

func NewMySQLMissionRepository(db *sql.DB) *MySQLMissionRepository {
	return &MySQLMissionRepository{
		db: db,
	}
}

func (m *MySQLMissionRepository) Add(ctx context.Context, mission models.Mission) (models.Mission, error) {
	return m.add(ctx, m.db, mission)
}

func (m *MySQLMissionRepository) AddWithTx(ctx context.Context, tx *sql.Tx, mission models.Mission) (models.Mission, error) {
	return m.add(ctx, tx, mission)
}

func (m *MySQLMissionRepository) add(ctx context.Context, querier Querier, mission models.Mission) (models.Mission, error) {
	newMissionQuery := `INSERT INTO missions () VALUES ()`
	result, err := querier.ExecContext(ctx, newMissionQuery)

	if err != nil {
		return models.Mission{}, err
	}
	mission.Id, err = result.LastInsertId()
	if err != nil {
		return models.Mission{}, err
	}
	return mission, nil
}

func (m *MySQLMissionRepository) WithTransaction(ctx context.Context, fn func(*sql.Tx) (models.Mission, error)) (models.Mission, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Mission{}, err
	}
	defer tx.Rollback()

	mission, err := fn(tx)
	if err != nil {
		return models.Mission{}, err
	}
	return mission, tx.Commit()
}

func (m *MySQLMissionRepository) GetById(ctx context.Context, id int64) (models.Mission, error) {
	var mission models.Mission
	getByIdQuery := `SELECT id, cat_id, completed FROM missions WHERE id = ? ORDER BY id`
	err := m.db.QueryRowContext(ctx, getByIdQuery, id).
		Scan(&mission.Id, &mission.CatId, &mission.Completed)
	if err != nil {
		return models.Mission{}, err
	}
	return mission, nil
}

func (m *MySQLMissionRepository) GetAll(ctx context.Context) ([]models.Mission, error) {
	var missions []models.Mission
	getAllQuery := `SELECT id, cat_id, completed FROM missions ORDER BY id`
	rows, err := m.db.QueryContext(ctx, getAllQuery)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		ms := new(models.Mission)
		if err := rows.Scan(&ms.Id, &ms.CatId, &ms.Completed); err != nil {
			return nil, err
		}
		missions = append(missions, *ms)
	}
	return missions, nil
}

func (m *MySQLMissionRepository) Assign(ctx context.Context, missionId, catId int64) error {
	assignMissionQuery := `UPDATE missions SET cat_id = ? WHERE id = ?`
	res, err := m.db.ExecContext(ctx, assignMissionQuery, catId, missionId)
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

func (m *MySQLMissionRepository) Complete(ctx context.Context, id int64) error {
	completeQuery := `UPDATE missions SET completed = ? where id = ?`
	res, err := m.db.ExecContext(ctx, completeQuery, true, id)
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

func (m *MySQLMissionRepository) Delete(ctx context.Context, id int64) error {
	deleteQuery := `DELETE FROM missions WHERE id = ?`
	res, err := m.db.ExecContext(ctx, deleteQuery, id)
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
