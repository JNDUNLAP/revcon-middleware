package mongo

import (
	"context"
	"dunlap/app/log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ValidateMongoKey(uri, databaseName, collectionName, providedAPIKey string) bool {
	log.Info("Validating Key in Mongo")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Error("Error connecting to MongoDB: %v", err)
		return false
	}
	defer client.Disconnect(ctx)

	collection := client.Database(databaseName).Collection(collectionName)

	filter := map[string]string{"apiKey": providedAPIKey}

	var result struct {
		APIKey string `bson:"apiKey"`
	}

	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Error("API key not found")
		} else {
			log.Error("Error querying MongoDB: %v", err)
		}
		return false
	}

	return result.APIKey == providedAPIKey
}
