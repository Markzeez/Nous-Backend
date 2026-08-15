package service

import (
	"context"
	"errors"
	"time"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type ChatService struct {
	chatRepo    *repository.ChatRepository
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
}

func NewChatService(chatRepo *repository.ChatRepository, userRepo *repository.UserRepository) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

func (s *ChatService) GetOrCreateRoom(ctx context.Context, patientID, professionalID string) (*models.ChatRoom, error) {
	room, err := s.chatRepo.GetByParticipants(ctx, patientID, professionalID)
	if err != nil {
		return nil, err
	}

	if room == nil {
		room = &models.ChatRoom{
			ID:             repository.GenerateID(),
			PatientID:      patientID,
			ProfessionalID: professionalID,
			IsActive:       true,
			CreatedAt:      time.Now(),
		}
		if err := s.chatRepo.Create(ctx, room); err != nil {
			return nil, errors.New("failed to create chat room")
		}
	}

	return room, nil
}

func (s *ChatService) GetMessages(ctx context.Context, roomID, userID string, page, limit int) ([]*models.Message, int64, error) {
	// Verify user is part of this room
	room, err := s.chatRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, 0, err
	}
	if room == nil {
		return nil, 0, errors.New("room not found")
	}

	if room.PatientID != userID && room.ProfessionalID != userID {
		return nil, 0, errors.New("access denied")
	}

	// This requires messageRepo - will be injected separately
	return nil, 0, nil
}

func (s *ChatService) ListRooms(ctx context.Context, userID string) ([]*models.ChatRoom, error) {
	return s.chatRepo.ListByUser(ctx, userID)
}
