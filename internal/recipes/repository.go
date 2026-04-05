package recipes

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
)

type scanner interface {
	Scan(dest ...any) error
}

const (
	recipesTable     = "recipes"
	brewStepsTable   = "brew_steps"
	ingredientsTable = "recipe_ingredients"
)

var (
	recipeSelectColumns = []string{
		"recipes.id",
		"recipes.author_id",
		"recipes.title",
		"recipes.description",
		"recipes.image_id",
		"recipes.brew_method",
		"recipes.difficulty",
		"recipes.roast_level",
		"recipes.beans",
		"recipes.note",
		"recipes.public",
		"recipes.created_at",
		"recipes.updated_at",
		"recipes.origin_recipe_id",
		"users.username",
		"users.avatar_id",
		"origin_recipes.author_id",
		"origin_authors.username",
		"origin_authors.avatar_id",
	}

	recipeInsertColumns = []string{
		"id",
		"author_id",
		"title",
		"description",
		"image_id",
		"brew_method",
		"difficulty",
		"roast_level",
		"beans",
		"note",
		"public",
		"created_at",
		"updated_at",
		"origin_recipe_id",
	}

	recipeSortableColumns = []string{
		"recipes.created_at",
		"recipes.title",
	}

	recipePreviewColumns = []string{
		"recipes.id",
		"recipes.author_id",
		"recipes.title",
		"recipes.image_id",
		"recipes.brew_method",
		"recipes.difficulty",
		"recipes.roast_level",
		"recipes.created_at",
		"recipes.origin_recipe_id",
		"users.username",
		"users.avatar_id",
		"origin_recipes.author_id",
		"origin_authors.username",
		"origin_authors.avatar_id",
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

	ingredientColumns = []string{
		"id",
		"recipe_id",
		"position",
		"name",
		"amount",
		"unit",
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

	var originRecipeID *string
	if recipe.Origin != nil {
		originRecipeID = &recipe.Origin.RecipeId
	}

	query := r.psql.Insert(recipesTable).
		Columns(recipeInsertColumns...).
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
			recipe.Note,
			recipe.Public,
			now,
			now,
			originRecipeID,
		).
		Suffix("ON CONFLICT (id) DO UPDATE SET " +
			"title = EXCLUDED.title, " +
			"description = EXCLUDED.description, " +
			"image_id = EXCLUDED.image_id, " +
			"brew_method = EXCLUDED.brew_method, " +
			"difficulty = EXCLUDED.difficulty, " +
			"roast_level = EXCLUDED.roast_level, " +
			"beans = EXCLUDED.beans, " +
			"note = EXCLUDED.note, " +
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
			q = q.Values(
				nil,
				recipe.Id,
				step.Order,
				step.Title,
				ptr.GetOr(step.Description, ""),
				step.DurationSeconds,
				step.ImageId,
			)
		}
		if _, err = q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert steps: %w", err)
		}
	}

	// ingredients
	_, err = r.psql.Delete(ingredientsTable).
		Where("recipe_id = ?", recipe.Id).
		RunWith(tx).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete ingredients: %w", err)
	}

	if len(recipe.Ingredients) > 0 {
		q := r.psql.Insert(ingredientsTable).Columns(ingredientColumns...)
		for i, ing := range recipe.Ingredients {
			q = q.Values(nil, recipe.Id, i, ing.Name, ing.Amount, ing.Unit)
		}
		if _, err = q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert ingredients: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Repository) GetRecipeByID(ctx context.Context, recipeID string) (models.Recipe, error) {
	sb := r.psql.
		Select(recipeSelectColumns...).
		From(recipesTable).
		Join("users ON users.id = recipes.author_id").
		LeftJoin("recipes AS origin_recipes ON origin_recipes.id = recipes.origin_recipe_id").
		LeftJoin("users AS origin_authors ON origin_authors.id = origin_recipes.author_id").
		Where(sq.Eq{"recipes.id": recipeID}).
		Limit(1)

	row := sb.RunWith(r.runner).QueryRowContext(ctx)

	recipe, err := scanRecipe(row)
	if err != nil {
		return models.Recipe{}, err
	}

	stepsMap, err := r.getBrewStepsByRecipeIDs(ctx, []string{recipeID})
	if err != nil {
		return models.Recipe{}, err
	}
	if steps, ok := stepsMap[recipeID]; ok {
		recipe.Steps = steps
	}

	ingredientsMap, err := r.getIngredientsByRecipeIDs(ctx, []string{recipeID})
	if err != nil {
		return models.Recipe{}, err
	}
	recipe.Ingredients = ingredientsMap[recipeID]
	if recipe.Ingredients == nil {
		recipe.Ingredients = []models.RecipeIngredient{}
	}

	return recipe, nil
}

func (r *Repository) ListRecipes(
	ctx context.Context, currentUserID string, params models.ListRecipesParams,
) (models.GenericPage[models.Recipe], error) {
	pag := params.Pagination()

	sb := r.psql.
		Select(recipeSelectColumns...).
		From(recipesTable).
		Join("users ON users.id = recipes.author_id").
		LeftJoin("recipes AS origin_recipes ON origin_recipes.id = recipes.origin_recipe_id").
		LeftJoin("users AS origin_authors ON origin_authors.id = origin_recipes.author_id")
	sb = applyListRecipesFilter(sb, params, currentUserID)
	sb = applyPagination(sb, pag)
	sb = applySort(sb, params.SortField, params.SortDirection, recipeSortableColumns, "recipes.created_at")

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

	ingredientsMap, err := r.getIngredientsByRecipeIDs(ctx, recipeIDs)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}

	for i := range recipes {
		recipes[i].Steps = stepsMap[recipes[i].Id]
		recipes[i].Ingredients = ingredientsMap[recipes[i].Id]
		if recipes[i].Ingredients == nil {
			recipes[i].Ingredients = []models.RecipeIngredient{}
		}
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

func (r *Repository) GetRecipesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Recipe, error) {
	rows, err := r.psql.
		Select(recipeSelectColumns...).
		From(recipesTable).
		Join("users ON users.id = recipes.author_id").
		LeftJoin("recipes AS origin_recipes ON origin_recipes.id = recipes.origin_recipe_id").
		LeftJoin("users AS origin_authors ON origin_authors.id = origin_recipes.author_id").
		Where(sq.Eq{"recipes.id": ids}).
		Where(sq.Or{
			sq.Eq{"recipes.public": true},
			sq.Eq{"recipes.author_id": currentUserID},
		}).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews by ids: %w", err)
	}
	defer rows.Close()

	var recipes []models.Recipe
	var recipeIDs []string
	for rows.Next() {
		recipe, err := scanRecipe(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recipe preview: %w", err)
		}
		recipes = append(recipes, recipe)
		recipeIDs = append(recipeIDs, recipe.Id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(recipes) > 0 {
		ingredientsMap, err := r.getIngredientsByRecipeIDs(ctx, recipeIDs)
		if err != nil {
			return nil, fmt.Errorf("get ingredients: %w", err)
		}
		for i := range recipes {
			recipes[i].Ingredients = ingredientsMap[recipes[i].Id]
			if recipes[i].Ingredients == nil {
				recipes[i].Ingredients = []models.RecipeIngredient{}
			}
		}
	}

	return recipes, nil
}

func (r *Repository) GetPreviewsByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.RecipePreview, error) {
	rows, err := r.psql.
		Select(recipePreviewColumns...).
		From(recipesTable).
		Join("users ON users.id = recipes.author_id").
		LeftJoin("recipes AS origin_recipes ON origin_recipes.id = recipes.origin_recipe_id").
		LeftJoin("users AS origin_authors ON origin_authors.id = origin_recipes.author_id").
		Where(sq.Eq{"recipes.id": ids}).
		Where(sq.Or{
			sq.Eq{"recipes.public": true},
			sq.Eq{"recipes.author_id": currentUserID},
		}).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews by ids: %w", err)
	}
	defer rows.Close()

	var previews []models.RecipePreview
	for rows.Next() {
		preview, err := scanRecipePreview(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recipe preview: %w", err)
		}
		previews = append(previews, preview)
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
		sb = sb.Where(sq.Eq{"recipes.brew_method": *params.BrewMethod})
	}

	// difficulty
	if params.Difficulty != nil {
		sb = sb.Where(sq.Eq{"recipes.difficulty": *params.Difficulty})
	}

	// roast level
	if params.RoastLevel != nil {
		sb = sb.Where(sq.Eq{"recipes.roast_level": *params.RoastLevel})
	}

	if params.Query != nil && *params.Query != "" {
		pattern := "%" + strings.ToLower(*params.Query) + "%"
		sb = sb.Where(sq.Or{
			sq.Expr("LOWER(recipes.title) LIKE ?", pattern),
			sq.Expr("LOWER(recipes.description) LIKE ?", pattern),
		})
	}

	// author filter
	if params.AuthorId != nil {
		authorID := *params.AuthorId
		sb = sb.Where(sq.Eq{"recipes.author_id": authorID})

		if authorID != currentUserID {
			sb = sb.Where(sq.Eq{"recipes.public": true})
		}
	} else {
		sb = sb.Where(sq.Eq{"recipes.public": true})
	}

	return sb
}

func applySort(
	sb sq.SelectBuilder,
	sortField *models.ListRecipesParamsSortField,
	sortDirection *models.SortDirection,
	allowedFields []string,
	defaultField string,
) sq.SelectBuilder {
	sort := defaultField

	if sortField != nil && slices.Contains(allowedFields, string(*sortField)) {
		sort = string(*sortField)
	}

	dir := "DESC"
	if sortDirection != nil {
		d := strings.ToUpper(string(*sortDirection))
		if d == "ASC" || d == "DESC" {
			dir = d
		}
	}

	return sb.OrderBy(fmt.Sprintf("%s %s, recipes.id %s", sort, dir, dir))
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
	var originRecipeID sql.NullString
	var originAuthorID sql.NullString
	var originUsername sql.NullString
	var originAvatarID sql.NullString

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
		&recipe.Note,
		&recipe.Public,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
		&originRecipeID,
		&recipe.Author.Username,
		&recipe.Author.AvatarId,
		&originAuthorID,
		&originUsername,
		&originAvatarID,
	)
	recipe.Author.Id = recipe.AuthorId

	if originRecipeID.Valid {
		recipe.Origin = &models.RecipeOrigin{
			RecipeId: originRecipeID.String,
			Author: models.UserPreview{
				Id:       originAuthorID.String,
				Username: originUsername.String,
				AvatarId: &originAvatarID.String,
			},
		}
	}

	return recipe, err
}

func scanRecipePreview(s scanner) (models.RecipePreview, error) {
	var p models.RecipePreview
	var originRecipeID sql.NullString
	var originAuthorID sql.NullString
	var originUsername sql.NullString
	var originAvatarID sql.NullString

	err := s.Scan(
		&p.Id, &p.AuthorId, &p.Title, &p.ImageId,
		&p.BrewMethod, &p.Difficulty, &p.RoastLevel,
		&p.CreatedAt,
		&originRecipeID,
		&p.Author.Username, &p.Author.AvatarId,
		&originAuthorID,
		&originUsername,
		&originAvatarID,
	)
	p.Author.Id = p.AuthorId

	if originRecipeID.Valid {
		p.Origin = &models.RecipeOrigin{
			RecipeId: originRecipeID.String,
			Author: models.UserPreview{
				Id:       originAuthorID.String,
				Username: originUsername.String,
				AvatarId: &originAvatarID.String,
			},
		}
	}

	return p, err
}

func (r *Repository) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	query := r.psql.Delete(recipesTable).Where(sq.And{
		sq.Eq{"author_id": userID},
		sq.Eq{"id": recipeID},
	})
	_, err := query.RunWith(r.runner).ExecContext(ctx)
	return err
}

func (r *Repository) getIngredientsByRecipeIDs(
	ctx context.Context,
	recipeIDs []string,
) (map[string][]models.RecipeIngredient, error) {
	rows, err := r.psql.
		Select(ingredientColumns...).
		From(ingredientsTable).
		Where(sq.Eq{"recipe_id": recipeIDs}).
		OrderBy("position ASC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.RecipeIngredient)
	for rows.Next() {
		var ing models.RecipeIngredient
		var recipeID string
		var id int64
		if err := rows.Scan(&id, &recipeID, new(int), &ing.Name, &ing.Amount, &ing.Unit); err != nil {
			return nil, err
		}
		result[recipeID] = append(result[recipeID], ing)
	}
	return result, rows.Err()
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
