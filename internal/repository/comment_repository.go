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

type CommentRepository struct {
	collection *mongo.Collection
}

func NewCommentRepository(mongoClient *db.MongoClient) *CommentRepository {
	return &CommentRepository{
		collection: mongoClient.Collection("comments"),
	}
}

func (r *CommentRepository) Create(ctx context.Context, comment *models.Comment) error {
	_, err := r.collection.InsertOne(ctx, comment)
	return err
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*models.Comment, error) {
	var comment models.Comment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

func (r *CommentRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *CommentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *CommentRepository) ListByPostID(ctx context.Context, postID string, page, limit int) ([]*models.Comment, int64, error) {
	filter := bson.M{"post_id": postID}
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

	var comments []*models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *CommentRepository) CountByPostID(ctx context.Context, postID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"post_id": postID})
}
