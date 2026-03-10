package recipe

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
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

func (r *Repository) CreateRecipe(ctx context.Context, recipe Recipe) error {
	query := r.psql.Insert("recipes").
		Columns("id", "author_id", "title", "description", "image_url", "brew_method", "difficulty", "roast_level", "beans").
		Values(recipe.ID, recipe.AuthorID, recipe.Title, recipe.Description, recipe.ImageURL, recipe.BrewMethod, recipe.Difficulty, recipe.RoastLevel, recipe.Beans)

	_, err := query.RunWith(r.db).ExecContext(ctx)
	if err != nil {
		return err
	}

	// brew steps
	for _, step := range recipe.Steps {
		_, err := r.psql.Insert("brew_steps").
			Columns("recipe_id", "step_order", "title", "description", "duration_seconds").
			Values(recipe.ID, step.Order, step.Title, step.Description, step.DurationSeconds).
			RunWith(r.db).ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) ListRecipes(ctx context.Context, params ListRecipesParams) (pagination.PaginatedResult[Recipe], error) {
	sb := r.psql.
		Select("id", "author_id", "title", "description", "image_url",
			"brew_method", "difficulty", "roast_level", "beans").
		From("recipes")

	conds := make(sq.Eq)
	if params.AuthorID != "" {
		conds["author_id"] = params.AuthorID
	}
	if params.BrewMethod != nil {
		conds["brew_method"] = *params.BrewMethod
	}
	if params.Difficulty != nil {
		conds["difficulty"] = *params.Difficulty
	}
	if len(conds) > 0 {
		sb = sb.Where(conds)
	}

	sb = sb.Limit(params.Pagination.Limit()).Offset(params.Pagination.Offset())
	rows, err := sb.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return pagination.PaginatedResult[Recipe]{}, err
	}
	defer rows.Close()

	recipes := make([]Recipe, 0)
	recipeIDs := make([]string, 0)
	for rows.Next() {
		var rcp Recipe
		if err := rows.Scan(
			&rcp.ID, &rcp.AuthorID, &rcp.Title, &rcp.Description,
			&rcp.ImageURL, &rcp.BrewMethod, &rcp.Difficulty,
			&rcp.RoastLevel, &rcp.Beans,
		); err != nil {
			return pagination.PaginatedResult[Recipe]{}, err
		}
		recipes = append(recipes, rcp)
		recipeIDs = append(recipeIDs, rcp.ID)
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
		return pagination.PaginatedResult[Recipe]{}, err
	}
	defer stepsRows.Close()

	stepsMap := make(map[string][]BrewStep)
	for stepsRows.Next() {
		var step BrewStep
		var recipeID string
		if err := stepsRows.Scan(&recipeID, &step.Order, &step.Title, &step.Description, &step.DurationSeconds); err != nil {
			return pagination.PaginatedResult[Recipe]{}, err
		}
		stepsMap[recipeID] = append(stepsMap[recipeID], step)
	}

	for i := range recipes {
		recipes[i].Steps = stepsMap[recipes[i].ID]
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
		return pagination.PaginatedResult[Recipe]{}, err
	}

	return pagination.NewResult(recipes, params.Pagination, total), nil
}

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete("recipes").Where(sq.Eq{"author_id": userID, "id": recipeID})
	_, err := query.RunWith(r.db).ExecContext(ctx)
	return err
}
