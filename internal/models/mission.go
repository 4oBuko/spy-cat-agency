package models

type Mission struct {
	Id        int64    `json:"id" db:"id"`
	CatId     int64    `json:"catId" db:"cat_id"`
	Targets   []Target `json:"targets" binding:"required,min=1"`
	Completed bool     `json:"completed" db:"completed"`
}
