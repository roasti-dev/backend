package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err.Error())
	}
}

func realMain() error {
	userID := flag.String("user", "test-user", "user ID to seed data for")
	entityType := flag.String("type", "", "entity type to seed (recipes)")
	filePath := flag.String("file", "", "path to JSON file (optional, uses embedded by default)")
	flag.Parse()

	if *entityType == "" {
		return fmt.Errorf("--type is required")
	}

	ctx := context.Background()

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "data.db"
	}

	database, err := db.NewSQLite(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("create db: %w", err)
	}
	defer database.Close() //nolint:errcheck

	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	uploadRepo := uploads.NewRepository(database)
	uploader := uploads.NewService(os.TempDir(), uploadRepo)
	recipeRepo := recipes.NewRepository(database, db.NewRunner(database, slog.Default(), false))
	likeRepo := likes.NewRepository(database)
	likeService := likes.NewService(likeRepo)
	recipeService := recipes.NewService(recipeRepo, uploader, likes.NewEnricher(likeService), likeService, nil, nil)

	switch *entityType {
	case "recipes":
		if err := seedRecipes(ctx, recipeService, *userID, *filePath); err != nil {
			return fmt.Errorf("seed recipes: %w", err)
		}
	default:
		return fmt.Errorf("unknown entity type: %s", *entityType)
	}

	slog.Info("Done")
	return nil
}
