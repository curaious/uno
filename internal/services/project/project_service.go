package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrProjectAlreadyExists = errors.New("project already exists")

// ProjectService contains business logic for projects
type ProjectService struct {
	repo *ProjectRepo
}

// NewProjectService constructs a new ProjectService
func NewProjectService(repo *ProjectRepo) *ProjectService {
	return &ProjectService{repo: repo}
}

// Create registers a new project ensuring name uniqueness
func (s *ProjectService) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("project name is required")
	}

	if _, err := s.repo.GetByName(ctx, req.Name); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrProjectAlreadyExists, req.Name)
	} else if !errors.Is(err, ErrProjectNotFound) {
		return nil, fmt.Errorf("failed to validate project name: %w", err)
	}

	project, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetByID fetches a project by its identifier
func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetByName fetches a project by its name
func (s *ProjectService) GetByName(ctx context.Context, name string) (*Project, error) {
	project, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// List returns all projects ordered by creation time
func (s *ProjectService) List(ctx context.Context) ([]*Project, error) {
	projects, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// Update modifies mutable project fields
func (s *ProjectService) Update(ctx context.Context, id uuid.UUID, req *UpdateProjectRequest) (*Project, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if req.Name != nil && *req.Name != existing.Name {
		if _, err := s.repo.GetByName(ctx, *req.Name); err == nil {
			return nil, fmt.Errorf("%w: %s", ErrProjectAlreadyExists, *req.Name)
		} else if !errors.Is(err, ErrProjectNotFound) {
			return nil, fmt.Errorf("failed to validate project name: %w", err)
		}
	}

	project, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

// Delete removes a project by ID
func (s *ProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}
