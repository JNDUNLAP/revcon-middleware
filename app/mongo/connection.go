package mongo

import (
	"context"
	"dunlap/app/log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func ConnectMongoDB(uri string) error {
	clientOptions := options.Client().ApplyURI(uri)

	c, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {

		return err
	}

	err = c.Ping(context.Background(), nil)
	if err != nil {
		return err
	}

	client = c
	log.Info("Connected to Mongo")
	return nil
}
