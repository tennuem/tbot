package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/tennuem/tbot/pkg/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoStore(addr string) (service.Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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
	var old service.Message
	merged := service.CloneMessage(m)
	err := collection.FindOne(ctx, bson.M{"cache_key": m.CacheKey}).Decode(&old)
	if err == nil {
		merged = service.MergeCache(&old, m)
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}
	_, err = collection.UpdateOne(ctx, bson.M{"cache_key": merged.CacheKey}, bson.M{
		"$set":         bson.M{"url": merged.URL, "provider": merged.Provider, "title": merged.Title, "links": merged.Links, "checked_at": merged.CheckedAt},
		"$setOnInsert": bson.M{"cache_key": merged.CacheKey, "user_id": merged.UserID, "user_name": merged.UserName, "chat_id": merged.ChatID, "first_seen_at": merged.FirstSeenAt},
	}, options.Update().SetUpsert(true))
	return err
}

func (s *mongoStore) FindByURL(ctx context.Context, host, url string) (*service.Message, error) {
	var m service.Message
	collection := s.client.Database("tbot").Collection("links")
	err := collection.FindOne(ctx, bson.M{"$or": []bson.M{
		{"provider": host, "url": url},
		{"links": bson.M{"$elemMatch": bson.M{"provider": host, "url": url}}},
	}}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, service.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *mongoStore) FindByUser(ctx context.Context, userID int) ([]service.Message, error) {
	var m []service.Message
	opts := options.Find()
	opts.SetLimit(10)
	collection := s.client.Database("tbot").Collection("links")
	cur, err := collection.Find(ctx, bson.D{{Key: "user_id", Value: userID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	if err := cur.All(ctx, &m); err != nil {
		return nil, err
	}
	return m, nil
}
