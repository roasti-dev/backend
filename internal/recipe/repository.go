package recipe

import (
	"context"
	"database/sql"

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
		Columns("id", "author_id", "title", "description", "image_url", "brew_method", "difficulty", "roast_level", "beans").
		Values(recipe.Id, recipe.AuthorId, recipe.Title, recipe.Description, recipe.ImageUrl, recipe.BrewMethod, recipe.Difficulty, recipe.RoastLevel, recipe.Beans)

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

func (r *Repository) ListRecipes(ctx context.Context, params ListRecipesParams) (pagination.PaginatedResult[models.Recipe], error) {
	sb := r.psql.
		Select("id", "author_id", "title", "description", "image_url",
			"brew_method", "difficulty", "roast_level", "beans").
		From("recipes")

	conds := make(sq.Eq)
	if params.AuthorID != "" {
		conds["author_id"] = params.AuthorID
	}
	if params.BrewMethod != nil && *params.BrewMethod != models.BrewMethodNone {
		conds["brew_method"] = params.BrewMethod
	}
	if params.Difficulty != nil && *params.Difficulty != models.DifficultyNone {
		conds["difficulty"] = params.Difficulty
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
			&rcp.RoastLevel, &rcp.Beans,
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
		OrderBy("recipe_id", "step_order").
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

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete("recipes").Where(sq.And{
		sq.Eq{"author_id": userID},
		sq.Eq{"id": recipeID},
	})
	_, err := query.RunWith(r.db).ExecContext(ctx)
	return err
}
