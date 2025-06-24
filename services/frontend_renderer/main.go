package main

import (
	"context"
	"html/template"
	"io"
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

// Template renderer
type Template struct {
	tmpl *template.Template
}

func loadTemplates() *Template {
	// Assume templates are in a 'views' directory relative to the binary
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
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

	// Renderer setup
	e.Renderer = loadTemplates()

	// Static files - assume 'css' directory relative to binary
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil) // Ensure correct template name
	})

	e.GET("/books", func(c echo.Context) error {
		books, err := findAllBooks(coll)
		if err != nil {
			log.Printf("Error in GET /books (findAllBooks): %v", err)
			return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to load books"})
		}
		return c.Render(http.StatusOK, "book-table.html", books) // Ensure correct template name
	})

	e.GET("/authors", func(c echo.Context) error {
		books, err := findAllBooks(coll)
		if err != nil {
			log.Printf("Error in GET /authors (findAllBooks): %v", err)
			return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to load authors"})
		}

		authorSet := make(map[string]struct{})
		for _, book := range books {
			if author, ok := book["author"].(string); ok && author != "" {
				authorSet[author] = struct{}{}
			}
		}
		authors := make([]string, 0, len(authorSet))
		for author := range authorSet {
			authors = append(authors, author)
		}
		return c.Render(http.StatusOK, "authors.html", map[string]interface{}{"Authors": authors})
	})

	e.GET("/years", func(c echo.Context) error {
		books, err := findAllBooks(coll)
		if err != nil {
			log.Printf("Error in GET /years (findAllBooks): %v", err)
			return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to load years"})
		}
		yearSet := make(map[string]struct{})
		for _, book := range books {
			if year, ok := book["year"].(string); ok {
				yearSet[year] = struct{}{}
			}
		}
		var years []string
		for year := range yearSet {
			years = append(years, year)
		}
		return c.Render(http.StatusOK, "years.html", map[string]interface{}{"Years": years})
	})

	// The original main.go had /search and /create, keeping /search for now.
	// /create returned NoContent, which might not be suitable for a frontend route.
	// If /search needs a specific template, it should be created.
	// For now, let's assume search.html exists or it's handled by client-side.
	e.GET("/search", func(c echo.Context) error {
		// Assuming search-bar.html is the correct template name from the original `main.go`
		return c.Render(http.StatusOK, "search-bar.html", nil)
	})
	// e.GET("/create", func(c echo.Context) error {
	// 	return c.NoContent(http.StatusNoContent)
	// })

	port := "3005"
	log.Printf("Frontend Renderer service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
