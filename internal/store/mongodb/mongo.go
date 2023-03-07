package mongodb

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

func (s *mongoStore) Save(ctx context.Context, m *service.Message) error {
	collection := s.client.Database("tbot").Collection("links")
	if _, err := collection.InsertOne(ctx, m); err != nil {
		return err
	}
	return nil
}

func (s *mongoStore) FindByURL(ctx context.Context, url string) (*service.Message, error) {
	var m service.Message
	collection := s.client.Database("tbot").Collection("links")
	if err := collection.FindOne(ctx, bson.D{{"url", url}}).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *mongoStore) FindByUsername(ctx context.Context, username string) ([]service.Message, error) {
	var m []service.Message
	opts := options.Find()
	opts.SetLimit(10)
	collection := s.client.Database("tbot").Collection("links")
	cur, err := collection.Find(ctx, bson.D{{"username", username}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	if err := cur.All(ctx, &m); err != nil {
		return nil, err
	}
	return m, nil
}
