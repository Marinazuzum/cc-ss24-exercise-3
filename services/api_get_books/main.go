package main

import (
	"context"
	"fmt"
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

// prepareData seeds the database with initial data if it doesn't exist
func prepareData(coll *mongo.Collection) {
	startData := []BookStore{
		{ID: "example1", BookName: "The Vortex", BookAuthor: "José Eustasio Rivera", BookEdition: "958-30-0804-4", BookPages: "292", BookYear: "1924"},
		{ID: "example2", BookName: "Frankenstein", BookAuthor: "Mary Shelley", BookEdition: "978-3-649-64609-9", BookPages: "280", BookYear: "1818"},
		{ID: "example3", BookName: "The Black Cat", BookAuthor: "Edgar Allan Poe", BookEdition: "978-3-99168-238-7", BookPages: "280", BookYear: "1843"},
	}

	for _, book := range startData {
		filter := bson.M{"ID": book.ID}
		count, err := coll.CountDocuments(context.TODO(), filter)
		if err != nil {
			log.Printf("Error counting documents for book ID %s: %v", book.ID, err)
			continue
		}
		if count == 0 {
			if _, err := coll.InsertOne(context.TODO(), book); err != nil {
				log.Printf("Error inserting book ID %s: %v", book.ID, err)
			} else {
				fmt.Printf("Inserted book: %+v\n", book.BookName)
			}
		} else {
			// fmt.Printf("Book ID %s already exists.\n", book.ID)
		}
	}
}

// findAllBooks retrieves all books from the collection
func findAllBooks(coll *mongo.Collection) ([]map[string]interface{}, error) {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"id":      res.ID,
			"title":   res.BookName,
			"author":  res.BookAuthor,
			"pages":   res.BookPages,
			"edition": res.BookEdition,
			"year":    res.BookYear,
		})
	}
	return ret, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Increased timeout
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

	// Ping the primary
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatalf("Failed to ping MongoDB: %v", err)
    }
    log.Println("Successfully connected and pinged MongoDB.")

	coll, err := prepareDatabase(client, "exercise-1", "information")
	if err != nil {
		log.Fatalf("Failed to prepare database: %v", err)
	}

	// It's usually better to run data seeding as a separate job or ensure idempotency.
	// For this exercise, running it on startup of the GET service is acceptable.
	prepareData(coll)

	e := echo.New()
	e.Use(middleware.Logger())

	e.GET("/api/books", func(c echo.Context) error {
		books, err := findAllBooks(coll)
		if err != nil {
			log.Printf("Error in GET /api/books (findAllBooks): %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return c.JSON(http.StatusOK, books)
	})

	e.GET("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		var result BookStore
		err := coll.FindOne(context.TODO(), bson.M{"ID": id}).Decode(&result)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.NoContent(http.StatusNotFound) // Changed to 404 Not Found
			}
			log.Printf("Error in GET /api/books/:id (FindOne): %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db error"})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":      result.ID,
			"title":   result.BookName,
			"author":  result.BookAuthor,
			"pages":   result.BookPages,
			"edition": result.BookEdition,
			"year":    result.BookYear,
		})
	})

	port := "3001"
	log.Printf("API Get Books service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
