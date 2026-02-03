package utils

func PageRange(current, last int) []int {
	const window = 2 // pages around current

	start := current - window
	end := current + window

	if start < 1 {
		start = 1
	}
	if end > last {
		end = last
	}

	pages := []int{}
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	return pages
}
