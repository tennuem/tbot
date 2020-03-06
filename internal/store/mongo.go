package store

import (
	"context"
	"time"

	"github.com/tennuem/tbot/pkg/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoStore(addr string) (service.Store, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(addr))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return &mongoStore{client}, nil
}

type mongoStore struct {
	client *mongo.Client
}

func (s *mongoStore) Save(ctx context.Context, m *service.Model) error {
	collection := s.client.Database("tbot").Collection("links")
	if _, err := collection.InsertOne(ctx, m); err != nil {
		return err
	}
	return nil
}

func (s *mongoStore) FindByMsg(ctx context.Context, msg string) *service.Model {
	var m service.Model
	collection := s.client.Database("tbot").Collection("links")
	if err := collection.FindOne(ctx, bson.D{{"msg", msg}}).Decode(&m); err != nil {
		return nil
	}
	return &m
}
