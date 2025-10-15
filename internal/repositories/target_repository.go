package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

var ErrTargetNotFound = errors.New("target not found")

type TargetRepository interface {
	Add(ctx context.Context, target models.Target) (models.Target, error)
	GetByMissionId(ctx context.Context, id int64) ([]models.Target, error)
	GetById(ctx context.Context, id int64) (models.Target, error)
	Complete(ctx context.Context, id int64) error
	Update(ctx context.Context, id int64, update models.TargetUpdate) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) error
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
		return models.Target{}, fmt.Errorf("failed to add new target: %w", err)
	}

	target.Id, err = result.LastInsertId()
	if err != nil {
		return models.Target{}, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return target, nil
}

func (m *MySQLTargetRepository) GetByMissionId(ctx context.Context, id int64) ([]models.Target, error) {
	var targets []models.Target
	getByMissionIdQuery := `SELECT id, mission_id, target_name, country, notes, completed FROM targets WHERE mission_id = ? ORDER BY id`
	rows, err := m.db.QueryContext(ctx, getByMissionIdQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets by mission: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		t := new(models.Target)
		if err := rows.Scan(&t.Id, &t.MissionId, &t.Name, &t.Country, &t.Notes, &t.Completed); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		targets = append(targets, *t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return targets, nil
}

func (m *MySQLTargetRepository) GetById(ctx context.Context, id int64) (models.Target, error) {
	var t models.Target
	getByIdQuery := `SELECT id, mission_id, target_name, country, notes, completed FROM targets WHERE id = ?`
	err := m.db.QueryRowContext(ctx, getByIdQuery, id).
		Scan(&t.Id, &t.MissionId, &t.Name, &t.Country, &t.Notes, &t.Completed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Target{}, ErrTargetNotFound
		}
		return models.Target{}, fmt.Errorf("failed to get target by id: %w", err)
	}
	return t, nil
}

func (m *MySQLTargetRepository) Complete(ctx context.Context, id int64) error {
	err := m.Exists(ctx, id)
	if err != nil {
		return err
	}

	completeQuery := `UPDATE targets SET completed = 1 WHERE id = ?`
	_, err = m.db.ExecContext(ctx, completeQuery, id)
	if err != nil {
		return fmt.Errorf("failed to complete target: %w", err)
	}

	return nil
}

func (m *MySQLTargetRepository) Update(ctx context.Context, id int64, update models.TargetUpdate) error {
	err := m.Exists(ctx, id)
	if err != nil {
		return err
	}

	updateQuery := `UPDATE targets SET notes = ? where id = ?`
	_, err = m.db.ExecContext(ctx, updateQuery, update.Notes, id)
	if err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}
	return nil
}

func (m *MySQLTargetRepository) Delete(ctx context.Context, id int64) error {
	err := m.Exists(ctx, id)
	if err != nil {
		return err
	}

	deleteQuery := `DELETE FROM targets WHERE id = ?`
	_, err = m.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	return nil
}

func (m *MySQLTargetRepository) Exists(ctx context.Context, id int64) error {
	var exists bool
	catExistsQuery := "SELECT EXISTS (SELECT 1 FROM targets WHERE id = ?)"
	err := m.db.QueryRowContext(ctx, catExistsQuery, id).Scan(&exists)

	if err != nil {
		return fmt.Errorf("existence check failed: %w", err)
	}

	if !exists {
		return ErrTargetNotFound
	}

	return nil
}
