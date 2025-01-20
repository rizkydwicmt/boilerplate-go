package database

type DirectionEnum string

const (
	ASC  DirectionEnum = "asc"
	DESC DirectionEnum = "desc"
)

func (e DirectionEnum) ToString() string {
	switch e {
	case ASC:
		return "asc"
	case DESC:
		return "desc"
	}
	return ""
}
