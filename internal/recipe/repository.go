package recipe

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
)

type scanner interface {
	Scan(dest ...any) error
}

const (
	recipesTable   = "recipes"
	brewStepsTable = "brew_steps"
)

var (
	recipeColumns = []string{
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
	}

	brewStepsColumns = []string{
		"recipe_id",
		"step_order",
		"title",
		"description",
		"duration_seconds",
	}
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
	query := r.psql.Insert(recipesTable).
		Columns(recipeColumns...).
		Values(
			recipe.Id,
			recipe.AuthorId,
			recipe.Title,
			recipe.Description,
			recipe.ImageUrl,
			recipe.BrewMethod,
			recipe.Difficulty,
			recipe.RoastLevel,
			recipe.Beans,
			recipe.Public,
		)

	_, err := query.RunWith(r.db).ExecContext(ctx)
	if err != nil {
		return err
	}

	// brew steps
	for _, step := range recipe.Steps {
		_, err := r.psql.Insert(brewStepsTable).
			Columns(brewStepsColumns...).
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
		Select(recipeColumns...).
		From(recipesTable).
		Where(sq.Eq{"id": recipeID}).
		Limit(1)

	row := sb.RunWith(r.db).QueryRowContext(ctx)

	recipe, err := scanRecipe(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	stepsMap, err := r.getBrewStepsByRecipeIDs(ctx, []string{recipeID})
	if err != nil {
		return models.Recipe{}, err
	}

	steps, ok := stepsMap[recipeID]
	if ok {
		recipe.Steps = steps
	}
	return recipe, nil
}

func (r *Repository) ListRecipes(
	ctx context.Context, currentUserID string, params models.ListRecipesParams,
) (pagination.Page[models.Recipe], error) {
	pag := params.Pagination()

	sb := r.psql.
		Select(recipeColumns...).
		From(recipesTable)
	sb = applyListRecipesFilter(sb, params, currentUserID)
	sb = applyPagination(sb, pag)

	rows, err := sb.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return pagination.Page[models.Recipe]{}, err
	}
	defer rows.Close()

	recipes, recipeIDs, err := scanRecipes(rows)
	if err != nil {
		return pagination.Page[models.Recipe]{}, err
	}

	if len(recipes) == 0 {
		return pagination.NewPage(recipes, pag, 0), nil
	}

	stepsMap, err := r.getBrewStepsByRecipeIDs(ctx, recipeIDs)
	if err != nil {
		return pagination.Page[models.Recipe]{}, err
	}

	for i := range recipes {
		recipes[i].Steps = stepsMap[recipes[i].Id]
	}

	countBuilder := r.psql.
		Select("COUNT(*)").
		From(recipesTable)
	countBuilder = applyListRecipesFilter(countBuilder, params, currentUserID)

	var total int
	if err := countBuilder.
		RunWith(r.db).
		QueryRowContext(ctx).
		Scan(&total); err != nil {
		return pagination.Page[models.Recipe]{}, err
	}
	return pagination.NewPage(recipes, pag, total), nil
}

func applyListRecipesFilter(
	sb sq.SelectBuilder,
	params models.ListRecipesParams,
	currentUserID string,
) sq.SelectBuilder {

	// brew method
	if params.BrewMethod != nil && *params.BrewMethod != models.BrewMethodNone {
		sb = sb.Where(sq.Eq{"brew_method": params.BrewMethod})
	}

	// difficulty
	if params.Difficulty != nil && *params.Difficulty != models.DifficultyNone {
		sb = sb.Where(sq.Eq{"difficulty": params.Difficulty})
	}

	// author filter
	if params.AuthorId != nil {
		authorID := *params.AuthorId
		sb = sb.Where(sq.Eq{"author_id": authorID})

		if authorID != currentUserID {
			sb = sb.Where(sq.Eq{"public": true})
		}
	} else {
		sb = sb.Where("(author_id = ? OR public = ?)", currentUserID, true)
	}

	return sb
}

func applyPagination(sb sq.SelectBuilder, pag pagination.Pagination) sq.SelectBuilder {
	return sb.Limit(uint64(pag.Limit())).Offset(uint64(pag.Offset()))
}

func scanRecipes(rows *sql.Rows) ([]models.Recipe, []string, error) {
	var recipes []models.Recipe
	var recipeIDs []string

	for rows.Next() {
		rcp, err := scanRecipe(rows)
		if err != nil {
			return nil, nil, err
		}
		recipes = append(recipes, rcp)
		recipeIDs = append(recipeIDs, rcp.Id)
	}

	return recipes, recipeIDs, nil
}

func scanRecipe(s scanner) (models.Recipe, error) {
	var recipe models.Recipe

	err := s.Scan(
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

	return recipe, err
}

func (r *Repository) getBrewStepsByRecipeIDs(
	ctx context.Context,
	recipeIDs []string,
) (map[string][]models.BrewStep, error) {
	stepsRows, err := r.psql.
		Select(brewStepsColumns...).
		From(brewStepsTable).
		Where(sq.Eq{"recipe_id": recipeIDs}).
		OrderBy("step_order ASC").
		RunWith(r.db).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer stepsRows.Close()

	stepsMap := make(map[string][]models.BrewStep)
	for stepsRows.Next() {
		var step models.BrewStep
		var recipeID string
		if err := stepsRows.Scan(&recipeID, &step.Order, &step.Title, &step.Description, &step.DurationSeconds); err != nil {
			return nil, err
		}
		stepsMap[recipeID] = append(stepsMap[recipeID], step)
	}

	return stepsMap, nil
}

func (r *Repository) UpdateRecipe(ctx context.Context, userID, recipeID string, request models.PatchRecipeRequest) (models.Recipe, error) {
	update := r.psql.Update(recipesTable).Where(sq.Eq{"id": recipeID})

	if request.Title != nil {
		update = update.Set("title", *request.Title)
	}
	if request.Description != nil {
		update = update.Set("description", *request.Description)
	}
	if request.ImageUrl != nil {
		update = update.Set("image_url", *request.ImageUrl)
	}
	if request.BrewMethod != nil {
		update = update.Set("brew_method", *request.BrewMethod)
	}
	if request.Difficulty != nil {
		update = update.Set("difficulty", *request.Difficulty)
	}
	if request.RoastLevel != nil {
		update = update.Set("roast_level", *request.RoastLevel)
	}
	if request.Beans != nil {
		update = update.Set("beans", *request.Beans)
	}
	if request.Public != nil {
		update = update.Set("public", *request.Public)
	}

	if _, err := update.RunWith(r.db).ExecContext(ctx); err != nil {
		return models.Recipe{}, err
	}

	return r.GetRecipeByID(ctx, recipeID)
}

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete(recipesTable).Where(sq.And{
		sq.Eq{"author_id": userID},
		sq.Eq{"id": recipeID},
	})
	_, err := query.RunWith(r.db).ExecContext(ctx)
	return err
}
