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

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(mongoClient *db.MongoClient) *UserRepository {
	return &UserRepository{
		collection: mongoClient.Collection("users"),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *UserRepository) List(ctx context.Context, filter bson.M, page, limit int, sortBy, sortOrder string) ([]*models.User, int64, error) {
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	sortDir := 1
	if sortOrder == "desc" {
		sortDir = -1
	}
	if sortBy == "" {
		sortBy = "created_at"
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

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) CountByRole(ctx context.Context, role models.Role) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"role": role})
}

func (r *UserRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}

func (r *UserRepository) GetVerifiedProfessionals(ctx context.Context, roleFilter string, specialty string) ([]*models.User, error) {
	filter := bson.M{
		"is_verified":  true,
		"is_available": true,
		"role":         bson.M{"$in": []string{"DOCTOR", "PHARMACIST", "NURSE", "LAB_SCIENTIST"}},
	}

	if roleFilter != "" {
		filter["role"] = roleFilter
	}

	if specialty != "" {
		filter["specialty"] = bson.M{"$regex": specialty, "$options": "i"}
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}
