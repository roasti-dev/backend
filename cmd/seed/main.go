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
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/seed"
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
	defer database.Close()

	if err := db.InitSchema(database); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	uploadRepo := uploads.NewRepository(database)
	uploader := uploads.NewService(os.TempDir(), uploadRepo)
	recipeRepo := recipe.NewRepository(database, slog.Default())
	likeRepo := likes.NewRepository(database)
	likeService := likes.NewService(database, likeRepo, recipeRepo)
	recipeService := recipe.NewService(recipeRepo, uploader, likeService)

	switch *entityType {
	case "recipes":
		if err := seed.SeedRecipes(ctx, recipeService, *userID, *filePath); err != nil {
			return fmt.Errorf("seed recipes: %w", err)
		}
	default:
		return fmt.Errorf("unknown entity type: %s", *entityType)
	}

	slog.Info("Done")
	return nil
}
