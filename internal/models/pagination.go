package models

type Pagination struct {
	PageSize   int `json:"pageSize"`
	Page       int `json:"page"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type PaginatedCats struct {
	Cats []Cat      `json:"cats"`
	Meta Pagination `json:"meta"`
}

type PaginatedMissions struct {
	Meta     Pagination `json:"meta"`
	Missions []Mission  `json:"missions"`
}

type PaginationQuery struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=5,max=50"`
}
