package service

import (
	"context"
	"errors"
	"time"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type AmbulanceService struct {
	ambulanceRepo *repository.AmbulanceRepository
	userRepo      *repository.UserRepository
}

func NewAmbulanceService(ambulanceRepo *repository.AmbulanceRepository, userRepo *repository.UserRepository) *AmbulanceService {
	return &AmbulanceService{
		ambulanceRepo: ambulanceRepo,
		userRepo:      userRepo,
	}
}

func (s *AmbulanceService) DispatchAmbulance(ctx context.Context, patientID string, req *models.AmbulanceRequest) (*models.AmbulanceDispatch, error) {
	// Verify patient exists
	patient, err := s.userRepo.GetByID(ctx, patientID)
	if err != nil {
		return nil, err
	}
	if patient == nil {
		return nil, errors.New("patient not found")
	}

	dispatch := &models.AmbulanceDispatch{
		ID:               repository.GenerateID(),
		PatientID:        patientID,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Severity:         req.Severity,
		Status:           "DISPATCHED",
		DispatchedAt:     time.Now(),
		EstimatedArrival: strPtr("8-12 minutes"),
	}

	if req.Address != "" {
		dispatch.Address = &req.Address
	}
	if req.Notes != "" {
		dispatch.Notes = &req.Notes
	}

	if err := s.ambulanceRepo.Create(ctx, dispatch); err != nil {
		return nil, errors.New("failed to dispatch ambulance")
	}

	dispatch.Patient = patient
	return dispatch, nil
}

func (s *AmbulanceService) ListDispatches(ctx context.Context, patientID string, page, limit int) ([]*models.AmbulanceDispatch, int64, error) {
	return s.ambulanceRepo.ListByPatient(ctx, patientID, page, limit)
}

func (s *AmbulanceService) UpdateDispatch(ctx context.Context, dispatchID, status string) (*models.AmbulanceDispatch, error) {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "COMPLETED" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if err := s.ambulanceRepo.Update(ctx, dispatchID, updates); err != nil {
		return nil, errors.New("failed to update dispatch")
	}

	return s.ambulanceRepo.GetByID(ctx, dispatchID)
}

func (s *AmbulanceService) ListAllDispatches(ctx context.Context, page, limit int) ([]*models.AmbulanceDispatch, int64, error) {
	return s.ambulanceRepo.ListAll(ctx, page, limit)
}

func strPtr(s string) *string {
	return &s
}
