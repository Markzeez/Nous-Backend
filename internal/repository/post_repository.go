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

type PostRepository struct {
	collection *mongo.Collection
}

func NewPostRepository(mongoClient *db.MongoClient) *PostRepository {
	return &PostRepository{
		collection: mongoClient.Collection("posts"),
	}
}

func (r *PostRepository) Create(ctx context.Context, post *models.Post) error {
	_, err := r.collection.InsertOne(ctx, post)
	return err
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*models.Post, error) {
	var post models.Post
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *PostRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *PostRepository) List(ctx context.Context, filter bson.M, page, limit int, sortBy, sortOrder string) ([]*models.Post, int64, error) {
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

	var posts []*models.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *PostRepository) IncrementViewCount(ctx context.Context, id string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"view_count": 1}})
	return err
}

func (r *PostRepository) IncrementReplyCount(ctx context.Context, id string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"reply_count": 1}})
	return err
}

func (r *PostRepository) DecrementReplyCount(ctx context.Context, id string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"reply_count": -1}})
	return err
}

func (r *PostRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}
