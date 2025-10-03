package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/4oBuko/spy-cat-agency/internal/models"
)

var ErrMissionNotFound = errors.New("mission not found")

type MissionRepository interface {
	Add(ctx context.Context, mission models.Mission) (models.Mission, error)
	GetById(ctx context.Context, id int64) (models.Mission, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.Mission, error)
	Assign(ctx context.Context, missionId, catId int64) error
	Complete(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) error
	GetCount(ctx context.Context) (int, error)
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
		return models.Mission{}, fmt.Errorf("mission insert failed: %w", err)
	}
	mission.Id, err = result.LastInsertId()
	if err != nil {
		return models.Mission{}, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return mission, nil
}

func (m *MySQLMissionRepository) WithTransaction(ctx context.Context, fn func(*sql.Tx) (models.Mission, error)) (models.Mission, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Mission{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	mission, err := fn(tx)
	if err != nil {
		// don't format error because fn should return formated error
		return models.Mission{}, err
	}
	err = tx.Commit()
	if err != nil {
		return models.Mission{}, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return mission, nil
}

func (m *MySQLMissionRepository) GetById(ctx context.Context, id int64) (models.Mission, error) {
	var mission models.Mission
	var tpCatId sql.NullInt64
	getByIdQuery := `SELECT id, cat_id, completed FROM missions WHERE id = ? ORDER BY id`
	err := m.db.QueryRowContext(ctx, getByIdQuery, id).
		Scan(&mission.Id, &tpCatId, &mission.Completed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Mission{}, ErrMissionNotFound
		}
		return models.Mission{}, fmt.Errorf("failed to get mission by id: %w", err)
	}
	if tpCatId.Valid {
		mission.CatId = tpCatId.Int64
	}
	return mission, nil
}

func (m *MySQLMissionRepository) GetAll(ctx context.Context, limit, offset int) ([]models.Mission, error) {
	var missions []models.Mission
	getAllQuery := `SELECT id, cat_id, completed FROM missions ORDER BY id LIMIT ? OFFSET ?`
	rows, err := m.db.QueryContext(ctx, getAllQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all missions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tpCatId sql.NullInt64
		ms := new(models.Mission)
		if err := rows.Scan(&ms.Id, &tpCatId, &ms.Completed); err != nil {
			return nil, fmt.Errorf("scan failed :%w", err)
		}
		if tpCatId.Valid {
			ms.CatId = tpCatId.Int64
		}
		missions = append(missions, *ms)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return missions, nil
}

func (m *MySQLMissionRepository) Assign(ctx context.Context, missionId, catId int64) error {
	err := m.Exists(ctx, missionId)
	if err != nil {
		return err
	}

	assignMissionQuery := `UPDATE missions SET cat_id = ? WHERE id = ?`
	_, err = m.db.ExecContext(ctx, assignMissionQuery, catId, missionId)
	if err != nil {
		return fmt.Errorf("failed to assign mission to a cat: %w", err)
	}
	return nil
}

func (m *MySQLMissionRepository) Complete(ctx context.Context, id int64) error {
	err := m.Exists(ctx, id)
	if err != nil {
		return err
	}

	completeQuery := `UPDATE missions SET completed = ? where id = ?`
	_, err = m.db.ExecContext(ctx, completeQuery, true, id)
	if err != nil {
		return fmt.Errorf("failed to complete mission: %w", err)
	}
	return nil

}

func (m *MySQLMissionRepository) Delete(ctx context.Context, id int64) error {
	err := m.Exists(ctx, id)
	if err != nil {
		return err
	}

	deleteQuery := `DELETE FROM missions WHERE id = ?`
	_, err = m.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete mission: %w", err)
	}
	return nil
}

func (m *MySQLMissionRepository) Exists(ctx context.Context, id int64) error {
	var exists bool
	ExistsQuery := `SELECT EXISTS(SELECT 1 FROM missions WHERE id = ?)`
	err := m.db.QueryRowContext(ctx, ExistsQuery, id).Scan(&exists)

	if err != nil {
		return fmt.Errorf("existence check failed: %w", err)
	}
	if !exists {
		return ErrMissionNotFound
	}
	return nil
}

func (m *MySQLMissionRepository) GetCount(ctx context.Context) (int, error) {
	var count int
	countQuery := "SELECT COUNT(*) FROM missions"
	err := m.db.QueryRowContext(ctx, countQuery).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count cats: %w", err)
	}
	return count, nil
}
