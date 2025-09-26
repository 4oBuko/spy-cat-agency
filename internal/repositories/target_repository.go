package repositories

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type TargetRepository interface {
	Add(ctx context.Context, target models.Target) (models.Target, error)
	GetByMissionId(ctx context.Context, id int64) ([]models.Target, error)
}

type TxTargetRepository interface {
	TargetRepository
	AddWithTx(ctx context.Context, tx *sql.Tx, target models.Target) (models.Target, error)
}

type MySQLTargetRepository struct {
	db *sql.DB
}

func NewMySQLTargetRepository(db *sql.DB) *MySQLTargetRepository {
	return &MySQLTargetRepository{
		db: db,
	}
}

func (m *MySQLTargetRepository) Add(ctx context.Context, target models.Target) (models.Target, error) {
	return m.add(ctx, m.db, target)
}

func (m *MySQLTargetRepository) AddWithTx(ctx context.Context, tx *sql.Tx, target models.Target) (models.Target, error) {
	return m.add(ctx, tx, target)
}

func (m *MySQLTargetRepository) add(ctx context.Context, querier Querier, target models.Target) (models.Target, error) {
	createTargetQuery := `INSERT INTO targets (mission_id, target_name, country, notes) VALUES (?, ?, ?, ?)`
	result, err := querier.ExecContext(ctx, createTargetQuery, target.MissionId, target.Name, target.Country, target.Notes)
	if err != nil {
		return models.Target{}, err
	}
	target.Id, err = result.LastInsertId()
	if err != nil {
		return models.Target{}, err
	}
	return target, nil
}

func (m *MySQLTargetRepository) GetByMissionId(ctx context.Context, id int64) ([]models.Target, error) {
	var targets []models.Target
	getByMissionIdQuery := `SELECT id, mission_id, target_name, country, notes, completed FROM targets WHERE mission_id = ?`
	rows, err := m.db.QueryContext(ctx, getByMissionIdQuery, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		t := new(models.Target)
		if err := rows.Scan(&t.Id, &t.MissionId, &t.Name, &t.Country, &t.Notes, &t.Completed); err != nil {
			return nil, err
		}
		targets = append(targets, *t)
	}
	return targets, nil
}
