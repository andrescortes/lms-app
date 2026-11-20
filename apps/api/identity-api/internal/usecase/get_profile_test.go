package usecase

import (
	"context"
	"errors"
	"testing"

	"lms-english/identity-service/internal/domain"
)

func TestGetProfileReturnsView(t *testing.T) {
	goal := "Pass B2"
	profiles := &profileStub{view: ProfileView{ID: "u1", Email: "a@b.com", Goals: &goal}}

	view, err := NewGetProfileUseCase(profiles).Execute(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if view.Goals == nil || *view.Goals != goal {
		t.Errorf("Goals = %v, want %q", view.Goals, goal)
	}

	missing := &profileStub{getErr: domain.ErrNotFound}
	if _, err := NewGetProfileUseCase(missing).Execute(context.Background(), "u1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestGetProfileHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := NewGetProfileUseCase(&profileStub{}).Execute(ctx, "u1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
