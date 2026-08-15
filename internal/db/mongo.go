package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient wraps the MongoDB client
type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoClient creates a new MongoDB client
func NewMongoClient(uri, dbName string) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Printf("[mongo] Connected to database: %s", dbName)
	return &MongoClient{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

// Collection returns a collection handle
func (m *MongoClient) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}
