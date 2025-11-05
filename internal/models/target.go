package models

type Target struct {
	Id        int64  `json:"id" db:"id"`
	MissionId int64  `json:"-" db:"mission_id"`
	Name      string `json:"name" db:"target_name" bindings:"required,min=1,max=50"`
	Country   string `json:"country" db:"country" bindings:"required,min=1,max=200"`
	Notes     string `json:"notes" db:"notes" bindings:"max=500"`
	Completed bool   `json:"completed" db:"completed"`
}

type TargetUpdate struct {
	Notes string `json:"notes"`
}
