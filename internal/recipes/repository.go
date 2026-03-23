package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
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
		"image_id",
		"brew_method",
		"difficulty",
		"roast_level",
		"beans",
		"public",
		"created_at",
		"updated_at",
		"likes_count",
	}

	recipeSortableColumns = []string{
		"created_at",
		"title",
	}

	recipePreviewColumns = []string{
		"id", "author_id", "title", "image_id",
		"brew_method", "difficulty", "roast_level",
		"likes_count", "created_at",
	}

	brewStepsColumns = []string{
		"id",
		"recipe_id",
		"step_order",
		"title",
		"description",
		"duration_seconds",
		"image_id",
	}
)

type Repository struct {
	db     *sql.DB
	runner sq.StdSqlCtx
	psql   sq.StatementBuilderType
}

func NewRepository(db *sql.DB, runner sq.StdSqlCtx) *Repository {
	return &Repository{
		runner: runner,
		db:     db,
		psql:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(runner),
	}
}

func (r *Repository) UpsertRecipe(ctx context.Context, recipe models.Recipe) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC()

	query := r.psql.Insert(recipesTable).
		Columns(recipeColumns...).
		Values(
			recipe.Id,
			recipe.AuthorId,
			recipe.Title,
			recipe.Description,
			recipe.ImageId,
			recipe.BrewMethod,
			recipe.Difficulty,
			recipe.RoastLevel,
			recipe.Beans,
			recipe.Public,
			now,
			now,
			0,
		).
		Suffix("ON CONFLICT (id) DO UPDATE SET " +
			"title = EXCLUDED.title, " +
			"description = EXCLUDED.description, " +
			"image_id = EXCLUDED.image_id, " +
			"brew_method = EXCLUDED.brew_method, " +
			"difficulty = EXCLUDED.difficulty, " +
			"roast_level = EXCLUDED.roast_level, " +
			"beans = EXCLUDED.beans, " +
			"public = EXCLUDED.public, " +
			"updated_at = EXCLUDED.updated_at")

	if _, err := query.RunWith(tx).ExecContext(ctx); err != nil {
		return fmt.Errorf("upsert recipe: %w", err)
	}

	// brew steps

	_, err = r.psql.Delete(brewStepsTable).
		Where("recipe_id = ?", recipe.Id).
		RunWith(tx).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete steps: %w", err)
	}

	if len(recipe.Steps) > 0 {
		q := r.psql.Insert(brewStepsTable).Columns(brewStepsColumns...)
		for _, step := range recipe.Steps {
			q = q.Values(nil, recipe.Id, step.Order, step.Title, ptr.GetOr(step.Description, ""), step.DurationSeconds, step.ImageId)
		}
		if _, err = q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert steps: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Repository) GetRecipeByID(ctx context.Context, recipeID string) (models.Recipe, error) {
	sb := r.psql.
		Select(recipeColumns...).
		From(recipesTable).
		Where(sq.Eq{"id": recipeID}).
		Limit(1)

	row := sb.RunWith(r.runner).QueryRowContext(ctx)

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
) (models.GenericPage[models.Recipe], error) {
	pag := params.Pagination()

	sb := r.psql.
		Select(recipeColumns...).
		From(recipesTable)
	sb = applyListRecipesFilter(sb, params, currentUserID)
	sb = applyPagination(sb, pag)
	sb = applySort(sb, params.SortField, params.SortDirection, recipeSortableColumns)

	rows, err := sb.RunWith(r.runner).QueryContext(ctx)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}
	defer rows.Close()

	recipes, recipeIDs, err := scanRecipes(rows)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}

	if len(recipes) == 0 {
		return models.NewPage(recipes, pag, 0), nil
	}

	stepsMap, err := r.getBrewStepsByRecipeIDs(ctx, recipeIDs)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
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
		RunWith(r.runner).
		QueryRowContext(ctx).
		Scan(&total); err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}
	return models.NewPage(recipes, pag, total), nil
}

func (r *Repository) GetPreviewsByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.RecipePreview, error) {
	rows, err := r.psql.
		Select(recipePreviewColumns...).
		From(recipesTable).
		Where(sq.Eq{"id": ids}).
		Where(sq.Or{
			sq.Eq{"public": true},
			sq.Eq{"author_id": currentUserID},
		}).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews by ids: %w", err)
	}
	defer rows.Close()

	var previews []models.RecipePreview
	for rows.Next() {
		var p models.RecipePreview
		if err := rows.Scan(
			&p.Id, &p.AuthorId, &p.Title, &p.ImageId,
			&p.BrewMethod, &p.Difficulty, &p.RoastLevel,
			&p.LikesCount, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recipe preview: %w", err)
		}
		previews = append(previews, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return previews, nil
}

func applyListRecipesFilter(
	sb sq.SelectBuilder,
	params models.ListRecipesParams,
	currentUserID string,
) sq.SelectBuilder {

	// brew method
	if params.BrewMethod != nil {
		sb = sb.Where(sq.Eq{"brew_method": *params.BrewMethod})
	}

	// difficulty
	if params.Difficulty != nil {
		sb = sb.Where(sq.Eq{"difficulty": *params.Difficulty})
	}

	if params.Query != nil && *params.Query != "" {
		pattern := "%" + *params.Query + "%"
		sb = sb.Where(sq.Or{
			sq.Like{"LOWER(title)": strings.ToLower(pattern)},
			sq.Like{"description": pattern},
		})
	}

	// author filter
	if params.AuthorId != nil {
		authorID := *params.AuthorId
		sb = sb.Where(sq.Eq{"author_id": authorID})

		if authorID != currentUserID {
			sb = sb.Where(sq.Eq{"public": true})
		}
	} else {
		sb = sb.Where(sq.Eq{"public": true})
	}

	return sb
}

func applySort(sb sq.SelectBuilder, sortField, sortDirection *string, allowedFields []string) sq.SelectBuilder {
	sort := "created_at"
	if sortField != nil && slices.Contains(allowedFields, *sortField) {
		sort = *sortField
	}

	dir := "DESC"
	if sortDirection != nil {
		d := strings.ToUpper(*sortDirection)
		if d == "ASC" || d == "DESC" {
			dir = d
		}
	}

	return sb.OrderBy(fmt.Sprintf("%s %s, id %s", sort, dir, dir))
}

func applyPagination(sb sq.SelectBuilder, pag models.PaginationParams) sq.SelectBuilder {
	return sb.Limit(uint64(pag.GetLimit())).Offset(uint64(pag.Offset()))
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
		&recipe.ImageId,
		&recipe.BrewMethod,
		&recipe.Difficulty,
		&recipe.RoastLevel,
		&recipe.Beans,
		&recipe.Public,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
		&recipe.LikesCount,
	)

	return recipe, err
}

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete(recipesTable).Where(sq.And{
		sq.Eq{"author_id": userID},
		sq.Eq{"id": recipeID},
	})
	_, err := query.RunWith(r.runner).ExecContext(ctx)
	return err
}

func (r *Repository) IncrementLikes(ctx context.Context, tx *sql.Tx, targetID string) (int, error) {
	var repoDb sq.BaseRunner
	if tx != nil {
		repoDb = tx
	} else {
		repoDb = r.runner
	}
	var count int
	err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(repoDb).
		Update("recipes").
		Set("likes_count", sq.Expr("likes_count + 1")).
		Where(sq.Eq{"id": targetID}).
		Suffix("RETURNING likes_count").
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, likes.ErrTargetNotFound
		}
		return 0, fmt.Errorf("increment likes: %w", err)
	}
	return count, nil
}

func (r *Repository) DecrementLikes(ctx context.Context, tx *sql.Tx, targetID string) (int, error) {
	var repoDb sq.BaseRunner
	if tx != nil {
		repoDb = tx
	} else {
		repoDb = r.runner
	}
	var count int
	err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(repoDb).
		Update("recipes").
		Set("likes_count", sq.Expr("MAX(likes_count - 1, 0)")).
		Where(sq.Eq{"id": targetID}).
		Suffix("RETURNING likes_count").
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, likes.ErrTargetNotFound
		}
		return 0, fmt.Errorf("decrement likes: %w", err)
	}
	return count, nil
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
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer stepsRows.Close()

	stepsMap := make(map[string][]models.BrewStep)
	for stepsRows.Next() {
		var step models.BrewStep
		var recipeID string
		if err := stepsRows.Scan(
			&step.Id, &recipeID, &step.Order, &step.Title, &step.Description, &step.DurationSeconds, &step.ImageId,
		); err != nil {
			return nil, err
		}
		stepsMap[recipeID] = append(stepsMap[recipeID], step)
	}

	return stepsMap, nil
}
