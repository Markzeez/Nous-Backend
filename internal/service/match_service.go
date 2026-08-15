package service

import (
	"context"
	"errors"
	"math"
	"time"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type MatchService struct {
	matchRepo      *repository.MatchRepository
	userRepo       *repository.UserRepository
	bookingService *BookingService
	chatService    *ChatService
	emailService   *EmailService
}

func NewMatchService(
	matchRepo *repository.MatchRepository,
	userRepo *repository.UserRepository,
	bookingService *BookingService,
	chatService *ChatService,
	emailService *EmailService,
) *MatchService {
	return &MatchService{
		matchRepo:      matchRepo,
		userRepo:       userRepo,
		bookingService: bookingService,
		chatService:    chatService,
		emailService:   emailService,
	}
}

var bookingTypeRoles = map[string][]string{
	"CONSULTATION":      {"DOCTOR"},
	"HOME_VISIT":        {"NURSE", "DOCTOR"},
	"SAMPLE_COLLECTION": {"LAB_SCIENTIST", "NURSE"},
	"PRESCRIPTION":      {"PHARMACIST", "DOCTOR"},
}

func (s *MatchService) CreateMatch(ctx context.Context, patientID string, req *models.MatchingRequest) (map[string]interface{}, error) {
	// Create match request
	match := &models.MatchRequest{
		ID:          repository.GenerateID(),
		PatientID:   patientID,
		Specialty:   req.Specialty,
		BookingType: req.BookingType,
		Urgency:     req.Urgency,
		Description: &req.Description,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.matchRepo.Create(ctx, match); err != nil {
		return nil, errors.New("failed to create match request")
	}

	// Find best matching professional
	roles, ok := bookingTypeRoles[req.BookingType]
	if !ok {
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{"status": "rejected"})
		return nil, errors.New("unknown booking type")
	}

	professionals, err := s.userRepo.GetVerifiedProfessionals(ctx, "", req.Specialty)
	if err != nil {
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{"status": "rejected"})
		return nil, errors.New("failed to find professionals")
	}

	// Filter by role
	var filtered []*models.User
	for _, p := range professionals {
		for _, r := range roles {
			if p.Role == models.Role(r) {
				filtered = append(filtered, p)
				break
			}
		}
	}

	if len(filtered) == 0 {
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{"status": "rejected"})
		return nil, errors.New("no available professionals found")
	}

	// Score and find best match
	best := s.findBestMatch(filtered, req)

	// Create or get chat room
	room, err := s.chatService.GetOrCreateRoom(ctx, patientID, best.ID)
	if err != nil {
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{"status": "rejected"})
		return nil, errors.New("failed to create chat room")
	}

	// Update match request
	s.matchRepo.Update(ctx, match.ID, map[string]interface{}{
		"status":          "matched",
		"assigned_pro_id": best.ID,
	})

	// Create booking
	bookingReq := &models.BookingCreateRequest{
		ProfessionalID: best.ID,
		Type:           req.BookingType,
		Notes:          req.Description,
		Address:        req.Address,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
	}

	booking, err := s.bookingService.CreateBooking(ctx, patientID, bookingReq)
	if err != nil {
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{"status": "rejected"})
		return nil, errors.New("failed to create booking")
	}

	return map[string]interface{}{
		"room":             room,
		"professional":     best,
		"booking":          booking,
		"match_request_id": match.ID,
	}, nil
}

// CheckPendingMatchesForDoctor checks if there are pending matches for a doctor's specialty
// and notifies the waiting patients
func (s *MatchService) CheckPendingMatchesForDoctor(ctx context.Context, doctor *models.User) error {
	if doctor.Specialty == nil || *doctor.Specialty == "" {
		return nil
	}

	// Find pending match requests for this specialty
	pendingMatches, err := s.matchRepo.GetPendingBySpecialty(ctx, *doctor.Specialty)
	if err != nil {
		return err
	}

	for _, match := range pendingMatches {
		// Get patient info
		patient, err := s.userRepo.GetByID(ctx, match.PatientID)
		if err != nil || patient == nil {
			continue
		}

		// Notify patient that a doctor is now available
		if s.emailService.IsConfigured() {
			go func(p *models.User, d *models.User, spec string) {
				_ = s.emailService.SendDoctorAvailableNotification(p.Email, p.Name, d.Name, spec)
			}(patient, doctor, *doctor.Specialty)
		}

		// Update match request status to notified
		s.matchRepo.Update(ctx, match.ID, map[string]interface{}{
			"status":           "notified",
			"assigned_pro_id":  doctor.ID,
		})
	}

	return nil
}

func (s *MatchService) findBestMatch(professionals []*models.User, req *models.MatchingRequest) *models.User {
	best := professionals[0]
	bestScore := s.scoreProfessional(best, req)

	for i := 1; i < len(professionals); i++ {
		score := s.scoreProfessional(professionals[i], req)
		if score > bestScore {
			best = professionals[i]
			bestScore = score
		}
	}
	return best
}

func (s *MatchService) scoreProfessional(pro *models.User, req *models.MatchingRequest) float64 {
	score := 100.0

	// Distance scoring
	if req.Latitude != nil && req.Longitude != nil && pro.Latitude != nil && pro.Longitude != nil {
		dist := haversine(*req.Latitude, *req.Longitude, *pro.Latitude, *pro.Longitude)
		score -= math.Min(dist*2, 60)
	}

	// Specialty match
	if req.Specialty != "" && pro.Specialty != nil {
		score += 20
	}

	// Urgency bonus
	if req.Urgency == "critical" && pro.Latitude != nil {
		score += 30
	}

	return score
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (s *MatchService) ListMatches(ctx context.Context, patientID string, page, limit int) ([]*models.MatchRequest, int64, error) {
	return s.matchRepo.ListByPatient(ctx, patientID, page, limit)
}
