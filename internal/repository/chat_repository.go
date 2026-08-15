package repository

import (
	"context"
	"errors"

	"medcon/internal/db"
	"medcon/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatRepository struct {
	collection *mongo.Collection
}

func NewChatRepository(mongoClient *db.MongoClient) *ChatRepository {
	return &ChatRepository{
		collection: mongoClient.Collection("chat_rooms"),
	}
}

func (r *ChatRepository) Create(ctx context.Context, room *models.ChatRoom) error {
	_, err := r.collection.InsertOne(ctx, room)
	return err
}

func (r *ChatRepository) GetByID(ctx context.Context, id string) (*models.ChatRoom, error) {
	var room models.ChatRoom
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&room)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

func (r *ChatRepository) GetByParticipants(ctx context.Context, patientID, professionalID string) (*models.ChatRoom, error) {
	var room models.ChatRoom
	err := r.collection.FindOne(ctx, bson.M{
		"patient_id":      patientID,
		"professional_id": professionalID,
		"is_active":       true,
	}).Decode(&room)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

func (r *ChatRepository) Update(ctx context.Context, id string, updates bson.M) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *ChatRepository) ListByUser(ctx context.Context, userID string) ([]*models.ChatRoom, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"patient_id": userID},
			{"professional_id": userID},
		},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []*models.ChatRoom
	if err := cursor.All(ctx, &rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}
