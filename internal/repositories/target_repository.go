package repositories

import (
	"context"
	"database/sql"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

type TargetRepository interface {
	Add(ctx context.Context, target models.Target) (models.Target, error)
	GetByMissionId(ctx context.Context, id int64) ([]models.Target, error)
	GetById(ctx context.Context, id int64) (models.Target, error)
	Complete(ctx context.Context, id int64) error
	Update(ctx context.Context, id int64, update models.TargetUpdate) error
	Delete(ctx context.Context, id int64) error
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
	getByMissionIdQuery := `SELECT id, mission_id, target_name, country, notes, completed FROM targets WHERE mission_id = ? ORDER BY id`
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

func (m *MySQLTargetRepository) GetById(ctx context.Context, id int64) (models.Target, error) {
	var t models.Target
	getByIdQuery := `SELECT id, mission_id, target_name, country, notes, completed FROM targets WHERE id = ?`
	var c bool
	err := m.db.QueryRowContext(ctx, getByIdQuery, id).
		Scan(&t.Id, &t.MissionId, &t.Name, &t.Country, &t.Notes, &c)
	if err != nil {
		return models.Target{}, err
	}
	t.Completed = c
	return t, nil
}

func (m *MySQLTargetRepository) Complete(ctx context.Context, id int64) error {
	completeQuery := `UPDATE targets SET completed = 1 WHERE id = ?`
	res, err := m.db.ExecContext(ctx, completeQuery, id)
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

func (m *MySQLTargetRepository) Update(ctx context.Context, id int64, update models.TargetUpdate) error {
	updateQuery := `UPDATE targets SET notes = ? where id = ?`
	res, err := m.db.ExecContext(ctx, updateQuery, update.Notes, id)
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

func (m *MySQLTargetRepository) Delete(ctx context.Context, id int64) error {
	deleteQuery := `DELETE FROM targets WHERE id = ?`
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
