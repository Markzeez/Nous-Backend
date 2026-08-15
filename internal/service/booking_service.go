package service

import (
	"context"
	"errors"
	"time"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type BookingService struct {
	bookingRepo *repository.BookingRepository
	userRepo    *repository.UserRepository
	chatRepo    *repository.ChatRepository
}

func NewBookingService(bookingRepo *repository.BookingRepository, userRepo *repository.UserRepository, chatRepo *repository.ChatRepository) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		userRepo:    userRepo,
		chatRepo:    chatRepo,
	}
}

func (s *BookingService) CreateBooking(ctx context.Context, patientID string, req *models.BookingCreateRequest) (*models.Booking, error) {
	// Verify professional exists
	professional, err := s.userRepo.GetByID(ctx, req.ProfessionalID)
	if err != nil {
		return nil, err
	}
	if professional == nil {
		return nil, errors.New("professional not found")
	}

	// Create or get chat room
	room, err := s.chatRepo.GetByParticipants(ctx, patientID, req.ProfessionalID)
	if err != nil {
		return nil, err
	}

	if room == nil {
		// Create new chat room
		room = &models.ChatRoom{
			ID:             repository.GenerateID(),
			PatientID:      patientID,
			ProfessionalID: req.ProfessionalID,
			IsActive:       true,
			CreatedAt:      time.Now(),
		}
		if err := s.chatRepo.Create(ctx, room); err != nil {
			return nil, errors.New("failed to create chat room")
		}
	}

	booking := &models.Booking{
		ID:             repository.GenerateID(),
		PatientID:      patientID,
		ProfessionalID: req.ProfessionalID,
		RoomID:         &room.ID,
		Type:           req.Type,
		Status:         "PENDING",
		CreatedAt:      time.Now(),
	}

	if req.ScheduledAt != "" {
		booking.ScheduledAt = &req.ScheduledAt
	}
	if req.Notes != "" {
		booking.Notes = &req.Notes
	}
	if req.Address != "" {
		booking.Address = &req.Address
	}
	if req.Latitude != nil {
		booking.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		booking.Longitude = req.Longitude
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, errors.New("failed to create booking")
	}

	// Populate relations
	patient, _ := s.userRepo.GetByID(ctx, patientID)
	if patient != nil {
		patient.PasswordHash = ""
		booking.Patient = patient
	}
	professional.PasswordHash = ""
	booking.Professional = professional
	booking.Room = room

	return booking, nil
}

func (s *BookingService) ListBookings(ctx context.Context, userID string, isPatient bool, page, limit int) ([]*models.Booking, int64, error) {
	return s.bookingRepo.ListByUser(ctx, userID, isPatient, page, limit)
}

func (s *BookingService) UpdateBooking(ctx context.Context, bookingID, professionalID, status string) (*models.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, errors.New("booking not found")
	}

	// Verify professional owns this booking
	if booking.ProfessionalID != professionalID {
		return nil, errors.New("not authorized to update this booking")
	}

	if err := s.bookingRepo.Update(ctx, bookingID, map[string]interface{}{"status": status}); err != nil {
		return nil, errors.New("failed to update booking")
	}

	return s.bookingRepo.GetByID(ctx, bookingID)
}
