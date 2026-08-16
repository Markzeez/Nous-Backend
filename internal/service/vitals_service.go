package service

import (
	"context"
	"errors"

	"medcon/internal/models"
	"medcon/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

type VitalsService struct {
	vitalsRepo *repository.VitalsRepository
	userRepo   *repository.UserRepository
}

func NewVitalsService(vitalsRepo *repository.VitalsRepository, userRepo *repository.UserRepository) *VitalsService {
	return &VitalsService{
		vitalsRepo: vitalsRepo,
		userRepo:   userRepo,
	}
}

func (s *VitalsService) CreateVitals(ctx context.Context, recordedByUserID string, req *models.VitalsCreateRequest) (*models.Vitals, error) {
	// Verify patient exists
	patient, err := s.userRepo.GetByID(ctx, req.PatientID)
	if err != nil {
		return nil, err
	}
	if patient == nil {
		return nil, errors.New("patient not found")
	}

	// Verify recorder exists
	recorder, err := s.userRepo.GetByID(ctx, recordedByUserID)
	if err != nil {
		return nil, err
	}
	if recorder == nil {
		return nil, errors.New("recorder not found")
	}

	// Create vitals record
	vitals := models.NewVitals(req.PatientID, recordedByUserID, recorder.Name, string(recorder.Role))

	// Set provided values
	vitals.Temperature = req.Temperature
	vitals.BloodPressureSystolic = req.BloodPressureSystolic
	vitals.BloodPressureDiastolic = req.BloodPressureDiastolic
	vitals.HeartRate = req.HeartRate
	vitals.RespiratoryRate = req.RespiratoryRate
	vitals.OxygenSaturation = req.OxygenSaturation
	vitals.Weight = req.Weight
	vitals.Height = req.Height
	vitals.BloodGlucose = req.BloodGlucose
	vitals.BloodGlucoseUnit = req.BloodGlucoseUnit
	vitals.PainLevel = req.PainLevel
	vitals.LevelOfConsciousness = req.LevelOfConsciousness
	vitals.Notes = req.Notes

	// Use provided recordedAt or default to now
	if req.RecordedAt != nil {
		vitals.RecordedAt = *req.RecordedAt
	}

	// Calculate BMI if weight and height provided
	vitals.CalculateBMI()

	if err := s.vitalsRepo.Create(ctx, vitals); err != nil {
		return nil, errors.New("failed to create vitals record")
	}

	// Populate patient info for response
	vitals.Patient = patient
	vitals.Patient.PasswordHash = ""

	return vitals, nil
}

func (s *VitalsService) GetVitals(ctx context.Context, id string) (*models.Vitals, error) {
	vitals, err := s.vitalsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if vitals == nil {
		return nil, errors.New("vitals record not found")
	}

	// Populate patient info
	patient, err := s.userRepo.GetByID(ctx, vitals.PatientID)
	if err == nil && patient != nil {
		patient.PasswordHash = ""
		vitals.Patient = patient
	}

	return vitals, nil
}

func (s *VitalsService) UpdateVitals(ctx context.Context, id string, req *models.VitalsUpdateRequest) (*models.Vitals, error) {
	vitals, err := s.vitalsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if vitals == nil {
		return nil, errors.New("vitals record not found")
	}

	updates := bson.M{}

	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}
	if req.BloodPressureSystolic != nil {
		updates["blood_pressure_systolic"] = *req.BloodPressureSystolic
	}
	if req.BloodPressureDiastolic != nil {
		updates["blood_pressure_diastolic"] = *req.BloodPressureDiastolic
	}
	if req.HeartRate != nil {
		updates["heart_rate"] = *req.HeartRate
	}
	if req.RespiratoryRate != nil {
		updates["respiratory_rate"] = *req.RespiratoryRate
	}
	if req.OxygenSaturation != nil {
		updates["oxygen_saturation"] = *req.OxygenSaturation
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Height != nil {
		updates["height"] = *req.Height
	}
	if req.BloodGlucose != nil {
		updates["blood_glucose"] = *req.BloodGlucose
	}
	if req.BloodGlucoseUnit != "" {
		updates["blood_glucose_unit"] = req.BloodGlucoseUnit
	}
	if req.PainLevel != nil {
		updates["pain_level"] = *req.PainLevel
	}
	if req.LevelOfConsciousness != "" {
		updates["level_of_consciousness"] = req.LevelOfConsciousness
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	// Recalculate BMI if weight or height changed
	if req.Weight != nil || req.Height != nil {
		weight := vitals.Weight
		height := vitals.Height
		if req.Weight != nil {
			weight = req.Weight
		}
		if req.Height != nil {
			height = req.Height
		}
		if weight != nil && height != nil && *height > 0 {
			heightM := *height / 100.0
			bmi := *weight / (heightM * heightM)
			updates["bmi"] = bmi
		}
	}

	if len(updates) > 0 {
		if err := s.vitalsRepo.Update(ctx, id, updates); err != nil {
			return nil, errors.New("failed to update vitals record")
		}
	}

	return s.GetVitals(ctx, id)
}

func (s *VitalsService) DeleteVitals(ctx context.Context, id string) error {
	vitals, err := s.vitalsRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if vitals == nil {
		return errors.New("vitals record not found")
	}

	return s.vitalsRepo.Delete(ctx, id)
}

func (s *VitalsService) ListVitals(ctx context.Context, query *models.VitalsListQuery) (*models.VitalsListResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}

	vitals, total, err := s.vitalsRepo.ListByPatient(
		ctx,
		query.PatientID,
		query.Page,
		query.Limit,
		query.SortBy,
		query.SortOrder,
		query.DateFrom,
		query.DateTo,
	)
	if err != nil {
		return nil, err
	}

	// Populate patient info for each vitals record
	for _, v := range vitals {
		patient, err := s.userRepo.GetByID(ctx, v.PatientID)
		if err == nil && patient != nil {
			patient.PasswordHash = ""
			v.Patient = patient
		}
	}

	return &models.VitalsListResponse{
		Data:  vitals,
		Total: total,
		Page:  query.Page,
		Limit: query.Limit,
	}, nil
}

func (s *VitalsService) GetLatestVitals(ctx context.Context, patientID string) (*models.Vitals, error) {
	vitals, err := s.vitalsRepo.GetLatestByPatient(ctx, patientID)
	if err != nil {
		return nil, err
	}
	if vitals == nil {
		return nil, nil
	}

	// Populate patient info
	patient, err := s.userRepo.GetByID(ctx, vitals.PatientID)
	if err == nil && patient != nil {
		patient.PasswordHash = ""
		vitals.Patient = patient
	}

	return vitals, nil
}

// ConvertVitalsToUpdateRequest converts a VitalsCreateRequest to VitalsUpdateRequest
func ConvertToUpdateRequest(req *models.VitalsCreateRequest) *models.VitalsUpdateRequest {
	return &models.VitalsUpdateRequest{
		Temperature:            req.Temperature,
		BloodPressureSystolic:  req.BloodPressureSystolic,
		BloodPressureDiastolic: req.BloodPressureDiastolic,
		HeartRate:              req.HeartRate,
		RespiratoryRate:        req.RespiratoryRate,
		OxygenSaturation:       req.OxygenSaturation,
		Weight:                 req.Weight,
		Height:                 req.Height,
		BloodGlucose:           req.BloodGlucose,
		BloodGlucoseUnit:       req.BloodGlucoseUnit,
		PainLevel:              req.PainLevel,
		LevelOfConsciousness:   req.LevelOfConsciousness,
		Notes:                  req.Notes,
	}
}
