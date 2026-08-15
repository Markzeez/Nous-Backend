package service

import (
	"context"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type UserService struct {
	userRepo     *repository.UserRepository
	matchService *MatchService
}

func NewUserService(userRepo *repository.UserRepository, matchService *MatchService) *UserService {
	return &UserService{
		userRepo:     userRepo,
		matchService: matchService,
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		user.PasswordHash = ""
	}
	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*models.User, error) {
	// Remove fields that shouldn't be updated directly
	delete(updates, "password_hash")
	delete(updates, "email")
	delete(updates, "id")
	delete(updates, "created_at")

	// Check if is_available is being changed to true
	wasAvailable := false
	willBeAvailable := false

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		wasAvailable = user.IsAvailable
	}

	if val, ok := updates["is_available"]; ok {
		willBeAvailable = val.(bool)
	}

	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return nil, err
	}

	// If professional just became available, check pending matches
	if !wasAvailable && willBeAvailable && s.matchService != nil && user != nil {
		// Check if user is a professional (doctor, nurse, etc.)
		if user.Role == models.RoleDoctor || user.Role == models.RoleNurse ||
			user.Role == models.RolePharmacist || user.Role == models.RoleLabScientist {
			go func() {
				_ = s.matchService.CheckPendingMatchesForDoctor(context.Background(), user)
			}()
		}
	}

	return s.userRepo.GetByID(ctx, userID)
}

func (s *UserService) ListUsers(ctx context.Context, filter map[string]interface{}, page, limit int, sortBy, sortOrder string) ([]*models.User, int64, error) {
	// Convert filter to bson.M
	filterBson := make(map[string]interface{})
	for k, v := range filter {
		filterBson[k] = v
	}

	users, total, err := s.userRepo.List(ctx, filterBson, page, limit, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// Remove password hashes
	for _, u := range users {
		u.PasswordHash = ""
	}

	return users, total, nil
}

func (s *UserService) GetVerifiedProfessionals(ctx context.Context, role, specialty string) ([]*models.User, error) {
	users, err := s.userRepo.GetVerifiedProfessionals(ctx, role, specialty)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		u.PasswordHash = ""
	}

	return users, nil
}

// SetMatchService sets the match service reference (for circular dependency)
func (s *UserService) SetMatchService(ms *MatchService) {
	s.matchService = ms
}

func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}
