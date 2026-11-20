package usecase

import (
	"context"
	"fmt"

	"lms-english/identity-service/internal/domain"
)

// ListUsersQuery is the admin list request. Empty filter fields mean no filter. A
// zero Page or Limit together with a false Has flag means the caller named none and
// takes the default; a zero with Has true is an explicit value that validation
// refuses.
type ListUsersQuery struct {
	Role      string
	CEFRLevel string
	Search    string
	Page      int
	Limit     int
	HasPage   bool
	HasLimit  bool
}

// UserPageResult is one page of the admin list plus the paging that produced it.
type UserPageResult struct {
	Items []UserSummary
	Total int
	Page  int
	Limit int
}

// ListUsersUseCase normalizes the admin list request, validates it, and reads one
// page.
type ListUsersUseCase struct {
	users UserLister
}

// NewListUsersUseCase wires the list use case with its port.
func NewListUsersUseCase(users UserLister) *ListUsersUseCase {
	return &ListUsersUseCase{users: users}
}

// Execute applies defaults, validates the request, and returns one page.
func (uc *ListUsersUseCase) Execute(ctx context.Context, query ListUsersQuery) (UserPageResult, error) {
	if err := ctx.Err(); err != nil {
		return UserPageResult{}, err
	}

	page, limit := query.Page, query.Limit
	if !query.HasPage {
		page = 1
	}
	if !query.HasLimit {
		limit = domain.DefaultUserPageSize
	}

	if err := domain.ValidatePagination(page, limit); err != nil {
		return UserPageResult{}, fmt.Errorf("list users: %w", err)
	}
	if err := domain.ValidateRoleFilter(query.Role); err != nil {
		return UserPageResult{}, fmt.Errorf("list users: %w", err)
	}

	filter := ListFilter{
		Role:      query.Role,
		CEFRLevel: query.CEFRLevel,
		Search:    domain.NormalizeSearch(query.Search),
		Limit:     limit,
		Offset:    (page - 1) * limit,
	}
	result, err := uc.users.ListUsers(ctx, filter)
	if err != nil {
		return UserPageResult{}, fmt.Errorf("list users: %w", err)
	}

	return UserPageResult{
		Items: result.Items,
		Total: result.Total,
		Page:  page,
		Limit: limit,
	}, nil
}
