package helpers

const PaginationLimit = 10

type Pagination struct {
	Page     int
	Limit    int
	Offset   int
	Total    int
	LastPage int
}

func NewPagination(page int, total int) Pagination {
	return Pagination{
		Page:     page,
		Limit:    PaginationLimit,
		Offset:   (page - 1) * PaginationLimit,
		Total:    total,
		LastPage: (total + PaginationLimit - 1) / PaginationLimit,
	}
}
