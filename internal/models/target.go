package models

type Target struct {
	Id        int    `json:"id" db:"id"`
	Name      string `json:"name" db:"target_name"`
	Country   string `json:"country" db:"country"`
	Notes     string `json:"notes" db:"notes"`
	Completed bool   `json:"completed" db:"completed"`
}
