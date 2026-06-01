package oplog

import (
	"context"
	"strings"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type MongoStore struct {
	client     *qmgo.Client
	collection *qmgo.Collection
}

func OpenMongoStore(ctx context.Context, uri, dbName, collectionName string) (*MongoStore, error) {
	uri = strings.TrimSpace(uri)
	dbName = strings.TrimSpace(dbName)
	collectionName = strings.TrimSpace(collectionName)
	if collectionName == "" {
		collectionName = "operation_logs"
	}
	client, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: uri})
	if err != nil {
		return nil, err
	}
	return &MongoStore{
		client:     client,
		collection: client.Database(dbName).Collection(collectionName),
	}, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close(ctx)
}

func (s *MongoStore) record(ctx context.Context, event Event) {
	if s == nil || s.collection == nil {
		return
	}
	normalizeEvent(&event)
	_, _ = s.collection.InsertOne(ctx, event)
}

func (s *MongoStore) List(ctx context.Context, q Query) ([]Event, error) {
	if s == nil || s.collection == nil {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	query := bson.M{}
	if q.WorkspaceID != "" {
		query["workspace_id"] = q.WorkspaceID
	}
	var items []Event
	err := s.collection.Find(ctx, query).Sort("-timestamp").Limit(int64(limit)).All(&items)
	return items, err
}

func normalizeEvent(event *Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Level == "" {
		event.Level = LevelInfo
	}
	if event.Source == "" {
		event.Source = "operation"
	}
	if event.Message == "" {
		event.Message = event.Action
	}
}
