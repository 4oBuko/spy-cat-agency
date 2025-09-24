package models

type Misson struct {
	Id        int64    `json:"id" db:"id"`
	CatName   string   `json:"catName" db:"cat_name"`
	Targets   []Target `json:"targets"`
	Completed bool     `json:"completed" db:"completed"`
}
