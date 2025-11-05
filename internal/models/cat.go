package models

type Cat struct {
	Id                int64  `json:"id" db:"id"`
	Name              string `json:"name" db:"cat_name" binding:"required,min=1,max=50"`
	YearsOfExperience int    `json:"yearsOfExperience" db:"years_of_experience" binding:"required,gte=0"`
	Breed             string `json:"breed" db:"breed" binding:"required,max=120"`
	Salary            int    `json:"salary" db:"salary" binding:"required,gte=0"`
}

type CatUpdate struct {
	Salary int `json:"salary" binding:"required,gte=0"`
}
