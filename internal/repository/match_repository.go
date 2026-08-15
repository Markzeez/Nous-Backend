package repository

import (
	"context"
	"errors"

	"medcon/internal/db"
	"medcon/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MatchRepository struct {
	collection *mongo.Collection
}

func NewMatchRepository(mongoClient *db.MongoClient) *MatchRepository {
	return &MatchRepository{
		collection: mongoClient.Collection("match_requests"),
	}
}

func (r *MatchRepository) Create(ctx context.Context, match *models.MatchRequest) error {
	_, err := r.collection.InsertOne(ctx, match)
	return err
}

func (r *MatchRepository) GetByID(ctx context.Context, id string) (*models.MatchRequest, error) {
	var match models.MatchRequest
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&match)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

func (r *MatchRepository) Update(ctx context.Context, id string, updates bson.M) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *MatchRepository) ListByPatient(ctx context.Context, patientID string, page, limit int) ([]*models.MatchRequest, int64, error) {
	filter := bson.M{"patient_id": patientID}
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var matches []*models.MatchRequest
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, 0, err
	}

	return matches, total, nil
}

// GetPendingBySpecialty returns pending match requests for a specific specialty
func (r *MatchRepository) GetPendingBySpecialty(ctx context.Context, specialty string) ([]*models.MatchRequest, error) {
	filter := bson.M{
		"status":          "pending",
		"specialty":       specialty,
		"assigned_pro_id": bson.M{"$exists": false},
	}
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: 1}}) // Oldest first

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var matches []*models.MatchRequest
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, err
	}

	return matches, nil
}
