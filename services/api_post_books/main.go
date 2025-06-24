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

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		log.Println("MONGO_URI not set, using default localhost URI")
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

	// POST /api/books
	e.POST("/api/books", func(c echo.Context) error {
		var req struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Author  string `json:"author"`
			Pages   string `json:"pages"`
			Edition string `json:"edition"`
			Year    string `json:"year"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}
		// Check for duplicates (id, title, author, year, pages)
		filter := bson.D{
			{"ID", req.ID},
		}
		count, err := coll.CountDocuments(context.TODO(), filter)
		if err != nil {
			log.Printf("Error in POST /api/books (CountDocuments): %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error checking duplicate ID"})
		}
		if count > 0 {
			return c.JSON(http.StatusConflict, map[string]string{"error": "duplicate entry for ID: " + req.ID})
		}
		book := BookStore{
			ID:          req.ID,
			BookName:    req.Title,
			BookAuthor:  req.Author,
			BookPages:   req.Pages,
			BookEdition: req.Edition,
			BookYear:    req.Year,
		}
		_, err = coll.InsertOne(context.TODO(), book)
		if err != nil {
			log.Printf("Error in POST /api/books (InsertOne): %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error inserting book"})
		}
		return c.JSON(http.StatusCreated, map[string]string{"message": "book created", "id": req.ID})
	})

	port := "3002"
	log.Printf("API Post Books service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
