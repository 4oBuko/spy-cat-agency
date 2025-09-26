package models

import "database/sql"

type Mission struct {
	Id        int64         `json:"id" db:"id"`
	CatId     sql.NullInt64 `json:"catId" db:"cat_id"`
	Targets   []Target      `json:"targets" binding:"required,min=1"`
	Completed bool          `json:"completed" db:"completed"`
}

func (m *Mission) GetCatId() int64 {
	if m.CatId.Valid {
		return m.CatId.Int64
	} else {
		return 0
	}
}

func (m *Mission) SetCatId(value int64) {
	m.CatId = sql.NullInt64{Int64: value, Valid: true}
}
