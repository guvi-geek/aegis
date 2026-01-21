package repository

import (
	"context"

	mongoInfra "github.com/RishiKendai/aegis/internal/infra/mongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository(client *mongoInfra.Client) *MongoRepository {
	return &MongoRepository{
		db: client.Database,
	}
}

func (r *MongoRepository) InsertOne(ctx context.Context, collection string, document interface{}, opts ...*options.InsertOneOptions) error {
	_, err := r.db.Collection(collection).InsertOne(ctx, document, opts...)
	return err
}

func (r *MongoRepository) FindOne(ctx context.Context, collection string, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return r.db.Collection(collection).FindOne(ctx, filter, opts...)
}

func (r *MongoRepository) FindMany(ctx context.Context, collection string, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return r.db.Collection(collection).Find(ctx, filter, opts...)
}

func (r *MongoRepository) CountDocuments(ctx context.Context, collection string, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return r.db.Collection(collection).CountDocuments(ctx, filter, opts...)
}

func (r *MongoRepository) GetCollection(collectionName string) *mongo.Collection {
	return r.db.Collection(collectionName)
}
