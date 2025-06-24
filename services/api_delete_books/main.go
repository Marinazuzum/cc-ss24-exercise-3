package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BookStore model
type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty"`
	ID          string             `bson:"ID"`
	BookName    string             `bson:"BookName"`
	BookAuthor  string             `bson:"BookAuthor"`
	BookEdition string             `bson:"BookEdition"`
	BookPages   string             `bson:"BookPages"`
	BookYear    string             `bson:"BookYear"`
}

// prepareDatabase initializes the database and collection
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)
	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Printf("Failed to create collection: %v", err)
			return nil, err
		}
	}
	coll := db.Collection(collecName)
	return coll, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	uri := os.Getenv("DATABASE_URI")
	if uri == "" {
		log.Println("DATABASE_URI not set, using default localhost URI")
		uri = "mongodb://localhost:27017/exercise-1?authSource=admin"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Successfully connected and pinged MongoDB.")

	coll, err := prepareDatabase(client, "exercise-1", "information")
	if err != nil {
		log.Fatalf("Failed to prepare database: %v", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())

	// DELETE /api/books/:id
	e.DELETE("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		res, err := coll.DeleteOne(context.TODO(), bson.M{"ID": id})
		if err != nil {
			log.Printf("Error in DELETE /api/books/:id (DeleteOne): %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error"})
		}
		if res.DeletedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"}) // Changed to 404
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "book deleted", "id": id})
	})

	port := "3004"
	log.Printf("API Delete Books service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
