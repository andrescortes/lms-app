package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// UserLister lists accounts for the admin list. It satisfies the use case port.
var _ usecase.UserLister = (*UserLister)(nil)

// UserLister implements usecase.UserLister over the paginated user queries.
type UserLister struct {
	db *sql.DB
}

// NewUserLister wraps db for the admin list.
func NewUserLister(db *sql.DB) *UserLister {
	return &UserLister{db: db}
}

// ListUsers returns one page of accounts and the total matching the filter. The page
// query and the count query are given the same arguments, so the total always
// describes the filter the page was drawn from.
func (s *UserLister) ListUsers(ctx context.Context, filter usecase.ListFilter) (usecase.UserPage, error) {
	args := filterArgs(filter)
	q := sqlc.New(s.db)

	count, err := q.CountUsers(ctx, sqlc.CountUsersParams{
		Role:      args.role,
		CefrLevel: args.cefrLevel,
		Search:    args.search,
	})
	if err != nil {
		return usecase.UserPage{}, fmt.Errorf("count users: %w", err)
	}

	rows, err := q.ListUsersPage(ctx, sqlc.ListUsersPageParams{
		Role:       args.role,
		CefrLevel:  args.cefrLevel,
		Search:     args.search,
		PageOffset: int64(filter.Offset),
		PageLimit:  int64(filter.Limit),
	})
	if err != nil {
		return usecase.UserPage{}, fmt.Errorf("list users: %w", err)
	}

	items := make([]usecase.UserSummary, 0, len(rows))
	for _, row := range rows {
		role, err := domain.ParseRole(row.Role)
		if err != nil {
			return usecase.UserPage{}, fmt.Errorf("list users: %w", err)
		}
		createdAt, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
		if err != nil {
			return usecase.UserPage{}, fmt.Errorf("list users: parse created at: %w", err)
		}
		items = append(items, usecase.UserSummary{
			ID:          row.ID,
			Email:       row.Email,
			DisplayName: row.DisplayName,
			Role:        role,
			CEFRLevel:   cefrLevelOf(row.CefrLevel),
			CreatedAt:   createdAt,
		})
	}

	return usecase.UserPage{Items: items, Total: int(count)}, nil
}

// filterArgs converts a filter into the nullable SQL arguments. An empty field
// becomes nil, which the queries read as "no filter on this column".
type listArgs struct {
	role      interface{}
	cefrLevel interface{}
	search    interface{}
}

func filterArgs(filter usecase.ListFilter) listArgs {
	return listArgs{
		role:      optionalArg(filter.Role),
		cefrLevel: optionalArg(filter.CEFRLevel),
		search:    optionalArg(filter.Search),
	}
}

// optionalArg maps an empty string to a SQL NULL.
func optionalArg(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
