package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title     string             `json:"title" bson:"title"`
	Completed bool               `json:"completed" bson:"completed"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type TodoStore interface {
	Create(context.Context, *Todo) error
	List(context.Context) ([]Todo, error)
	Get(context.Context, primitive.ObjectID) (*Todo, error)
	Update(context.Context, primitive.ObjectID, bson.M) (*Todo, error)
	Delete(context.Context, primitive.ObjectID) error
}

type MongoTodoStore struct {
	collection *mongo.Collection
}

func NewMongoTodoStore(collection *mongo.Collection) *MongoTodoStore {
	return &MongoTodoStore{collection: collection}
}

func (s *MongoTodoStore) Create(ctx context.Context, todo *Todo) error {
	now := time.Now().UTC()
	todo.ID = primitive.NewObjectID()
	todo.CreatedAt, todo.UpdatedAt = now, now
	_, err := s.collection.InsertOne(ctx, todo)
	return err
}

func (s *MongoTodoStore) List(ctx context.Context) ([]Todo, error) {
	cursor, err := s.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	todos := make([]Todo, 0)
	if err := cursor.All(ctx, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *MongoTodoStore) Get(ctx context.Context, id primitive.ObjectID) (*Todo, error) {
	var todo Todo
	if err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *MongoTodoStore) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) (*Todo, error) {
	fields["updatedAt"] = time.Now().UTC()
	result := s.collection.FindOneAndUpdate(
		ctx, bson.M{"_id": id}, bson.M{"$set": fields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var todo Todo
	if err := result.Decode(&todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *MongoTodoStore) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
