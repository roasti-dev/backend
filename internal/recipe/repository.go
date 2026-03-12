package recipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
)

type Repository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *Repository) CreateRecipe(ctx context.Context, recipe models.Recipe) error {
	query := r.psql.Insert("recipes").
		Columns("id", "author_id", "title", "description", "image_url", "brew_method", "difficulty", "roast_level", "beans", "public").
		Values(recipe.Id, recipe.AuthorId, recipe.Title, recipe.Description, recipe.ImageUrl, recipe.BrewMethod, recipe.Difficulty, recipe.RoastLevel, recipe.Beans, recipe.Public)

	_, err := query.RunWith(r.db).ExecContext(ctx)
	if err != nil {
		return err
	}

	// brew steps
	for _, step := range recipe.Steps {
		_, err := r.psql.Insert("brew_steps").
			Columns("recipe_id", "step_order", "title", "description", "duration_seconds").
			Values(recipe.Id, step.Order, step.Title, step.Description, step.DurationSeconds).
			RunWith(r.db).ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) GetRecipeByID(ctx context.Context, recipeID string) (models.Recipe, error) {
	sb := r.psql.
		Select(
			"id",
			"author_id",
			"title",
			"description",
			"image_url",
			"brew_method",
			"difficulty",
			"roast_level",
			"beans",
			"public",
		).
		From("recipes").
		Where(sq.Eq{"id": recipeID}).
		Limit(1)

	row := sb.RunWith(r.db).QueryRowContext(ctx)

	var recipe models.Recipe
	err := row.Scan(
		&recipe.Id,
		&recipe.AuthorId,
		&recipe.Title,
		&recipe.Description,
		&recipe.ImageUrl,
		&recipe.BrewMethod,
		&recipe.Difficulty,
		&recipe.RoastLevel,
		&recipe.Beans,
		&recipe.Public,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, fmt.Errorf("recipe not found: %w", err)
		}
		return models.Recipe{}, err
	}

	stepsRows, err := r.psql.
		Select("step_order", "title", "description", "duration_seconds").
		From("brew_steps").
		Where(sq.Eq{"recipe_id": recipeID}).
		OrderBy("step_order").
		RunWith(r.db).
		QueryContext(ctx)
	if err != nil {
		return models.Recipe{}, err
	}
	defer stepsRows.Close()

	for stepsRows.Next() {
		var step models.BrewStep
		if err := stepsRows.Scan(&step.Order, &step.Title, &step.Description, &step.DurationSeconds); err != nil {
			return models.Recipe{}, err
		}
		recipe.Steps = append(recipe.Steps, step)
	}

	return recipe, nil
}

func (r *Repository) ListRecipes(ctx context.Context, params ListRecipesParams, currentUserID string) (pagination.PaginatedResult[models.Recipe], error) {
	sb := r.psql.
		Select("id", "author_id", "title", "description", "image_url",
			"brew_method", "difficulty", "roast_level", "beans", "public").
		From("recipes")

	conds := make(sq.Eq)

	if params.BrewMethod != nil && *params.BrewMethod != models.BrewMethodNone {
		conds["brew_method"] = params.BrewMethod
	}
	if params.Difficulty != nil && *params.Difficulty != models.DifficultyNone {
		conds["difficulty"] = params.Difficulty
	}

	if params.AuthorID != nil {
		authorID := *params.AuthorID
		conds["author_id"] = authorID

		if authorID != currentUserID {
			conds["public"] = true
		}
	} else {
		sb = sb.Where("(author_id = ? OR public = ?)", currentUserID, true)
	}

	if len(conds) > 0 {
		sb = sb.Where(conds)
	}

	sb = sb.Limit(params.Pagination.Limit()).Offset(params.Pagination.Offset())
	rows, err := sb.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return pagination.PaginatedResult[models.Recipe]{}, err
	}
	defer rows.Close()

	recipes := make([]models.Recipe, 0)
	recipeIDs := make([]string, 0)
	for rows.Next() {
		var rcp models.Recipe
		if err := rows.Scan(
			&rcp.Id, &rcp.AuthorId, &rcp.Title, &rcp.Description,
			&rcp.ImageUrl, &rcp.BrewMethod, &rcp.Difficulty,
			&rcp.RoastLevel, &rcp.Beans, &rcp.Public,
		); err != nil {
			return pagination.PaginatedResult[models.Recipe]{}, err
		}
		recipes = append(recipes, rcp)
		recipeIDs = append(recipeIDs, rcp.Id)
	}

	if len(recipes) == 0 {
		return pagination.NewResult(recipes, params.Pagination, 0), nil
	}

	stepsRows, err := r.psql.
		Select("recipe_id", "step_order", "title", "description", "duration_seconds").
		From("brew_steps").
		Where(sq.Eq{"recipe_id": recipeIDs}).
		OrderBy("step_order").
		RunWith(r.db).
		QueryContext(ctx)
	if err != nil {
		return pagination.PaginatedResult[models.Recipe]{}, err
	}
	defer stepsRows.Close()

	stepsMap := make(map[string][]models.BrewStep)
	for stepsRows.Next() {
		var step models.BrewStep
		var recipeID string
		if err := stepsRows.Scan(&recipeID, &step.Order, &step.Title, &step.Description, &step.DurationSeconds); err != nil {
			return pagination.PaginatedResult[models.Recipe]{}, err
		}
		stepsMap[recipeID] = append(stepsMap[recipeID], step)
	}

	for i := range recipes {
		recipes[i].Steps = stepsMap[recipes[i].Id]
	}

	var total int64
	err = r.psql.
		Select("COUNT(*)").
		From("recipes").
		Where(conds).
		RunWith(r.db).
		QueryRowContext(ctx).
		Scan(&total)
	if err != nil {
		return pagination.PaginatedResult[models.Recipe]{}, err
	}

	return pagination.NewResult(recipes, params.Pagination, total), nil
}

func (r *Repository) UpdateRecipe(ctx context.Context, userID, recipeID string, params UpdateRecipeParams) (models.Recipe, error) {
	update := r.psql.Update("recipes").Where(sq.Eq{"id": recipeID})

	if params.Title != nil {
		update = update.Set("title", *params.Title)
	}
	if params.Description != nil {
		update = update.Set("description", *params.Description)
	}
	if params.ImageURL != nil {
		update = update.Set("image_url", *params.ImageURL)
	}
	if params.BrewMethod != nil {
		update = update.Set("brew_method", *params.BrewMethod)
	}
	if params.Difficulty != nil {
		update = update.Set("difficulty", *params.Difficulty)
	}
	if params.RoastLevel != nil {
		update = update.Set("roast_level", *params.RoastLevel)
	}
	if params.Beans != nil {
		update = update.Set("beans", *params.Beans)
	}
	if params.Public != nil {
		update = update.Set("public", *params.Public)
	}

	// // если нет изменений, возвращаем текущий рецепт
	// if len(update.) == 0 {
	// 	return r.GetRecipeByID(ctx, recipeID)
	// }

	if _, err := update.RunWith(r.db).ExecContext(ctx); err != nil {
		return models.Recipe{}, err
	}

	// возвращаем свежий рецепт
	return r.GetRecipeByID(ctx, recipeID)
}

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete("recipes").Where(sq.And{
		sq.Eq{"author_id": userID},
		sq.Eq{"id": recipeID},
	})
	_, err := query.RunWith(r.db).ExecContext(ctx)
	return err
}
