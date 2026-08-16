package repository

import (
	"context"
	"errors"
	"time"

	"medcon/internal/db"
	"medcon/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VitalsRepository struct {
	collection *mongo.Collection
}

func NewVitalsRepository(mongoClient *db.MongoClient) *VitalsRepository {
	return &VitalsRepository{
		collection: mongoClient.Collection("vitals"),
	}
}

func (r *VitalsRepository) Create(ctx context.Context, vitals *models.Vitals) error {
	_, err := r.collection.InsertOne(ctx, vitals)
	return err
}

func (r *VitalsRepository) GetByID(ctx context.Context, id string) (*models.Vitals, error) {
	var vitals models.Vitals
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&vitals)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &vitals, nil
}

func (r *VitalsRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *VitalsRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *VitalsRepository) ListByPatient(ctx context.Context, patientID string, page, limit int, sortBy, sortOrder string, dateFrom, dateTo *time.Time) ([]*models.Vitals, int64, error) {
	filter := bson.M{"patient_id": patientID}

	// Add date range filter
	if dateFrom != nil || dateTo != nil {
		dateFilter := bson.M{}
		if dateFrom != nil {
			dateFilter["$gte"] = *dateFrom
		}
		if dateTo != nil {
			dateFilter["$lte"] = *dateTo
		}
		filter["recorded_at"] = dateFilter
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	sortDir := 1
	if sortOrder == "desc" {
		sortDir = -1
	}
	if sortBy == "" {
		sortBy = "recorded_at"
	}
	opts.SetSort(bson.D{{Key: sortBy, Value: sortDir}})

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var vitalsList []*models.Vitals
	if err := cursor.All(ctx, &vitalsList); err != nil {
		return nil, 0, err
	}

	return vitalsList, total, nil
}

func (r *VitalsRepository) GetLatestByPatient(ctx context.Context, patientID string) (*models.Vitals, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "recorded_at", Value: -1}})
	var vitals models.Vitals
	err := r.collection.FindOne(ctx, bson.M{"patient_id": patientID}, opts).Decode(&vitals)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &vitals, nil
}

func (r *VitalsRepository) CountByPatient(ctx context.Context, patientID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"patient_id": patientID})
}
