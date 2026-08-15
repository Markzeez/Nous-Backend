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

type BookingRepository struct {
	collection *mongo.Collection
}

func NewBookingRepository(mongoClient *db.MongoClient) *BookingRepository {
	return &BookingRepository{
		collection: mongoClient.Collection("bookings"),
	}
}

func (r *BookingRepository) Create(ctx context.Context, booking *models.Booking) error {
	_, err := r.collection.InsertOne(ctx, booking)
	return err
}

func (r *BookingRepository) GetByID(ctx context.Context, id string) (*models.Booking, error) {
	var booking models.Booking
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&booking)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &booking, nil
}

func (r *BookingRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *BookingRepository) ListByUser(ctx context.Context, userID string, isPatient bool, page, limit int) ([]*models.Booking, int64, error) {
	filter := bson.M{}
	if isPatient {
		filter["patient_id"] = userID
	} else {
		filter["professional_id"] = userID
	}

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

	var bookings []*models.Booking
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, 0, err
	}

	return bookings, total, nil
}
