package models

type Mission struct {
	Id        int64    `json:"id" db:"id"`
	CatId     int64    `json:"catId" db:"cat_id" binding:"required,gte=0"`
	Targets   []Target `json:"targets"`
	Completed bool     `json:"completed" db:"completed"`
}
