package database

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaginationResult struct {
	CurrentPage int         `json:"currentPage"`
	PerPage     int         `json:"perPage"`
	TotalItems  int64       `json:"totalItems"`
	TotalPages  int         `json:"totalPages"`
	Data        interface{} `json:"data"`
}

type CursorResult struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"nextCursor"`
	HasMore    bool        `json:"hasMore"`
	PerPage    int         `json:"perPage"`
}

type OrderField struct {
	Field     string
	Direction DirectionEnum
}

func (o OrderField) ToString() string {
	return fmt.Sprintf("%s %s", o.Field, o.Direction)
}

type PaginationQuery struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

func NewPaginationRequest(c *gin.Context) *PaginationQuery {
	var query PaginationQuery
	_ = c.ShouldBindQuery(&query)
	return query.Parse()
}

func (q *PaginationQuery) Parse() *PaginationQuery {
	page := q.Page
	if page <= 0 {
		page = defaultPage
	}

	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return &PaginationQuery{
		Page:  page,
		Limit: limit,
	}
}

func (q *PaginationQuery) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		qry := q.Parse()

		offset := (qry.Page - 1) * qry.Limit
		return db.Offset(offset).Limit(qry.Limit)
	}
}

// FindWithPagination executes the query with pagination and returns PaginationResult
//
// Example basic usage:
//
// var users []User
// pagination := database.NewPaginationRequest(c)
// result, err := db.FindWithPagination(pagination, &users)
// // result.Data contains first 10 users
// // result.TotalItems contains total count of users
//
// Example with conditions:
//
// var users []User
// pagination := database.NewPaginationRequest(c)
// db = db.Where("name LIKE ?", "%john%")
// result, err := db.FindWithPagination(pagination, &users)
// // result.Data contains first 10 users with name containing "john"
func (db *Database) FindWithPagination(query PaginationQuery, dest interface{}, conditions ...interface{}) (*PaginationResult, error) {
	var totalItems int64

	if err := db.Model(dest).Count(&totalItems).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(query.Limit)))

	if err := db.Scopes(query.Paginate()).Find(dest, conditions...).Error; err != nil {
		return nil, err
	}

	return &PaginationResult{
		CurrentPage: query.Page,
		PerPage:     query.Limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		Data:        dest,
	}, nil
}

// FindWithCursor executes the query with infinite scrolling and returns CursorResult
//
// Example basic usage:
//
//	var users []User
//	result, err := db.FindWithCursor("", 10, &users, "id")
//	// result.Items contains first 10 users
//	// result.NextCursor contains the cursor for the next page
//	// result.HasMore is true if there are more items
func (db *Database) FindWithCursor(encryptedCursor string, limit int, dest interface{}, order OrderField) (*CursorResult, error) {
	if limit <= 0 {
		limit = 10
	}

	limit++
	query := db.DB

	if encryptedCursor != "" {
		cursor, err := db.cursorCrypto.decrypt(encryptedCursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		if cursor != "" {
			query = query.Where(order.Field+" < ?", cursor)
		}
	}

	query = query.Order(order.ToString()).Limit(limit)

	if err := query.Find(dest).Error; err != nil {
		return nil, err
	}

	result := &CursorResult{
		Items:   dest,
		PerPage: limit - 1,
	}

	items := reflect.ValueOf(dest).Elem()
	if items.Len() == limit {
		items.Set(items.Slice(0, items.Len()-1))
		result.HasMore = true

		lastItem := items.Index(items.Len() - 1)
		cursorField := lastItem.FieldByName(cases.Title(language.Und).String(strings.Split(order.Field, ".")[len(strings.Split(order.Field, "."))-1]))
		nextCursor, err := db.cursorCrypto.encrypt(fmt.Sprint(cursorField.Interface()))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt cursor: %w", err)
		}
		result.NextCursor = nextCursor
	}

	return result, nil
}
