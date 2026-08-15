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

type AmbulanceRepository struct {
	collection *mongo.Collection
}

func NewAmbulanceRepository(mongoClient *db.MongoClient) *AmbulanceRepository {
	return &AmbulanceRepository{
		collection: mongoClient.Collection("ambulance_dispatches"),
	}
}

func (r *AmbulanceRepository) Create(ctx context.Context, dispatch *models.AmbulanceDispatch) error {
	_, err := r.collection.InsertOne(ctx, dispatch)
	return err
}

func (r *AmbulanceRepository) GetByID(ctx context.Context, id string) (*models.AmbulanceDispatch, error) {
	var dispatch models.AmbulanceDispatch
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&dispatch)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &dispatch, nil
}

func (r *AmbulanceRepository) Update(ctx context.Context, id string, updates bson.M) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *AmbulanceRepository) ListByPatient(ctx context.Context, patientID string, page, limit int) ([]*models.AmbulanceDispatch, int64, error) {
	filter := bson.M{"patient_id": patientID}
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "dispatched_at", Value: -1}})

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var dispatches []*models.AmbulanceDispatch
	if err := cursor.All(ctx, &dispatches); err != nil {
		return nil, 0, err
	}

	return dispatches, total, nil
}

func (r *AmbulanceRepository) ListAll(ctx context.Context, page, limit int) ([]*models.AmbulanceDispatch, int64, error) {
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "dispatched_at", Value: -1}})

	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var dispatches []*models.AmbulanceDispatch
	if err := cursor.All(ctx, &dispatches); err != nil {
		return nil, 0, err
	}

	return dispatches, total, nil
}
