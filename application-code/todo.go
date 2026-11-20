package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrTodoNotFound = errors.New("todo not found")

type Todo struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Completed   bool               `json:"completed" bson:"completed"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type TodoRepository interface {
	List(ctx context.Context) ([]Todo, error)
	Create(ctx context.Context, todo Todo) (Todo, error)
	Get(ctx context.Context, id primitive.ObjectID) (Todo, error)
	Update(ctx context.Context, id primitive.ObjectID, todo Todo) (Todo, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	Ping(ctx context.Context) error
}

type MongoTodoRepository struct {
	collection *mongo.Collection
}

func NewMongoTodoRepository(collection *mongo.Collection) *MongoTodoRepository {
	return &MongoTodoRepository{collection: collection}
}

func (r *MongoTodoRepository) List(ctx context.Context) ([]Todo, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
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

func (r *MongoTodoRepository) Create(ctx context.Context, todo Todo) (Todo, error) {
	result, err := r.collection.InsertOne(ctx, todo)
	if err != nil {
		return Todo{}, err
	}
	if id, ok := result.InsertedID.(primitive.ObjectID); ok {
		todo.ID = id
	}
	return todo, nil
}

func (r *MongoTodoRepository) Get(ctx context.Context, id primitive.ObjectID) (Todo, error) {
	var todo Todo
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&todo)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Todo{}, ErrTodoNotFound
	}
	return todo, err
}

func (r *MongoTodoRepository) Update(ctx context.Context, id primitive.ObjectID, todo Todo) (Todo, error) {
	update := bson.M{"$set": bson.M{
		"title":       todo.Title,
		"description": todo.Description,
		"completed":   todo.Completed,
		"updatedAt":   todo.UpdatedAt,
	}}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return Todo{}, err
	}
	if result.MatchedCount == 0 {
		return Todo{}, ErrTodoNotFound
	}
	return r.Get(ctx, id)
}

func (r *MongoTodoRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrTodoNotFound
	}
	return nil
}

func (r *MongoTodoRepository) Ping(ctx context.Context) error {
	return r.collection.Database().Client().Ping(ctx, nil)
}
