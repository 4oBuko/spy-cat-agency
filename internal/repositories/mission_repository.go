package repositories

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type MissionRepository interface {
	Add(ctx context.Context, mission models.Mission) (models.Mission, error)
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
	newMissionQuery := `INSERT INTO missions(cat_id) VALUES (?)`
	result, err := querier.ExecContext(ctx, newMissionQuery, mission.CatId)
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
