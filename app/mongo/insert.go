package mongo

import (
	"context"
	"dunlap/app/log"
)

func InsertDocument(ctx context.Context, dbName, collectionName string, document interface{}) error {
	database := client.Database(dbName)
	collection := database.Collection(collectionName)

	_, err := collection.InsertOne(ctx, document)
	if err != nil {
		return err
	}

	log.Info("Document inserted successfully")
	return nil
}
