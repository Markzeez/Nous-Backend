package repository

import (
	"context"

	"medcon/internal/db"
	"medcon/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepository struct {
	collection *mongo.Collection
}

func NewMessageRepository(mongoClient *db.MongoClient) *MessageRepository {
	return &MessageRepository{
		collection: mongoClient.Collection("messages"),
	}
}

func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	_, err := r.collection.InsertOne(ctx, message)
	return err
}

func (r *MessageRepository) GetByRoomID(ctx context.Context, roomID string, page, limit int) ([]*models.Message, int64, error) {
	filter := bson.M{"room_id": roomID}
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: 1}})

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *MessageRepository) MarkAsRead(ctx context.Context, roomID, userID string) error {
	_, err := r.collection.UpdateMany(ctx, bson.M{
		"room_id":   roomID,
		"sender_id": bson.M{"$ne": userID},
		"is_read":   false,
	}, bson.M{"$set": bson.M{"is_read": true}})
	return err
}
