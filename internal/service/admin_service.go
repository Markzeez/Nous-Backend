package service

import (
	"context"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type AdminService struct {
	userRepo      *repository.UserRepository
	postRepo      *repository.PostRepository
	commentRepo   *repository.CommentRepository
	bookingRepo   *repository.BookingRepository
	matchRepo     *repository.MatchRepository
	ambulanceRepo *repository.AmbulanceRepository
}

func NewAdminService(
	userRepo *repository.UserRepository,
	postRepo *repository.PostRepository,
	commentRepo *repository.CommentRepository,
	bookingRepo *repository.BookingRepository,
	matchRepo *repository.MatchRepository,
	ambulanceRepo *repository.AmbulanceRepository,
) *AdminService {
	return &AdminService{
		userRepo:      userRepo,
		postRepo:      postRepo,
		commentRepo:   commentRepo,
		bookingRepo:   bookingRepo,
		matchRepo:     matchRepo,
		ambulanceRepo: ambulanceRepo,
	}
}

func (s *AdminService) GetStats(ctx context.Context) (*models.AdminStatsResponse, error) {
	totalUsers, _ := s.userRepo.Count(ctx, map[string]interface{}{})
	totalPatients, _ := s.userRepo.CountByRole(ctx, models.RolePatient)
	totalDoctors, _ := s.userRepo.CountByRole(ctx, models.RoleDoctor)
	totalPharmacists, _ := s.userRepo.CountByRole(ctx, models.RolePharmacist)
	totalNurses, _ := s.userRepo.CountByRole(ctx, models.RoleNurse)
	totalLabScientists, _ := s.userRepo.CountByRole(ctx, models.RoleLabScientist)
	totalAdmins, _ := s.userRepo.CountByRole(ctx, models.RoleAdmin)

	totalPosts, _ := s.postRepo.Count(ctx, map[string]interface{}{})

	// For bookings and ambulance, we'd need to add count methods to those repos
	// For now, return 0
	return &models.AdminStatsResponse{
		TotalUsers:         int(totalUsers),
		TotalPatients:      int(totalPatients),
		TotalDoctors:       int(totalDoctors),
		TotalPharmacists:   int(totalPharmacists),
		TotalNurses:        int(totalNurses),
		TotalLabScientists: int(totalLabScientists),
		TotalAdmins:        int(totalAdmins),
		TotalPosts:         int(totalPosts),
		TotalBookings:      0,
		TotalAmbulance:     0,
	}, nil
}

func (s *AdminService) ListUsers(ctx context.Context, page, limit int) ([]*models.User, int64, error) {
	users, total, err := s.userRepo.List(ctx, map[string]interface{}{}, page, limit, "created_at", "desc")
	if err != nil {
		return nil, 0, err
	}

	for _, u := range users {
		u.PasswordHash = ""
	}

	return users, total, nil
}

func (s *AdminService) GetUser(ctx context.Context, id string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user != nil {
		user.PasswordHash = ""
	}
	return user, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, id string, req *models.AdminUserUpdateRequest) (*models.User, error) {
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.IsVerified != nil {
		updates["is_verified"] = *req.IsVerified
	}
	if req.IsAvailable != nil {
		updates["is_available"] = *req.IsAvailable
	}
	if req.Specialty != "" {
		updates["specialty"] = req.Specialty
	}
	if req.LicenseNumber != "" {
		updates["license_number"] = req.LicenseNumber
	}

	if err := s.userRepo.Update(ctx, id, updates); err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(ctx, id)
}

func (s *AdminService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.Delete(ctx, id)
}

func (s *AdminService) ListPosts(ctx context.Context, page, limit int) ([]*models.Post, int64, error) {
	posts, total, err := s.postRepo.List(ctx, map[string]interface{}{}, page, limit, "created_at", "desc")
	if err != nil {
		return nil, 0, err
	}

	// Populate authors
	for _, post := range posts {
		author, err := s.userRepo.GetByID(ctx, post.AuthorID)
		if err == nil && author != nil {
			author.PasswordHash = ""
			post.Author = author
		}
	}

	return posts, total, nil
}

func (s *AdminService) UpdatePost(ctx context.Context, id string, updates map[string]interface{}) (*models.Post, error) {
	if err := s.postRepo.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.postRepo.GetByID(ctx, id)
}

func (s *AdminService) DeletePost(ctx context.Context, id string) error {
	return s.postRepo.Delete(ctx, id)
}
