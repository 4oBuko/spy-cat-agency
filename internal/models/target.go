package models

type Target struct {
	Id        int64  `json:"id" db:"id"`
	MissionId int64  `json:"-" db:"mission_id"`
	Name      string `json:"name" db:"target_name"`
	Country   string `json:"country" db:"country"`
	Notes     string `json:"notes" db:"notes"`
	Completed bool   `json:"completed" db:"completed"`
}
