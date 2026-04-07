package beans

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
)

const beansTable = "beans"

var beanSelectColumns = []string{
	"b.id", "b.name", "b.roast_type", "b.roaster",
	"b.country", "b.region", "b.farm", "b.process",
	"b.descriptors", "b.q_score", "b.url", "b.image_id",
	"b.created_at",
	"b.author_id", "u.username", "u.avatar_id",
}

type scanner interface {
	Scan(dest ...any) error
}

type Repository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewRepository(db *sql.DB, runner sq.StdSqlCtx) *Repository {
	return &Repository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(runner),
	}
}

func (r *Repository) Create(ctx context.Context, beanID, authorID string, req models.BeanPayload) error {
	descriptorsJSON, err := marshalDescriptors(req.Descriptors)
	if err != nil {
		return err
	}
	_, err = r.psql.Insert(beansTable).
		Columns(
			"id", "author_id", "name", "roast_type", "roaster",
			"country", "region", "farm", "process",
			"descriptors", "q_score", "url", "image_id", "created_at",
		).
		Values(
			beanID, authorID, req.Name, req.RoastType, req.Roaster,
			req.Country, req.Region, req.Farm, req.Process,
			descriptorsJSON, req.QScore, req.Url, req.ImageId, time.Now().UTC(),
		).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert bean: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, beanID string) (models.Bean, error) {
	row := r.psql.
		Select(beanSelectColumns...).
		From(beansTable + " b").
		Join("users u ON u.id = b.author_id").
		Where(sq.Eq{"b.id": beanID}).
		Where("b.deleted_at IS NULL").
		QueryRowContext(ctx)

	bean, err := scanBean(row)
	if err != nil {
		return models.Bean{}, fmt.Errorf("get bean by id: %w", err)
	}
	return bean, nil
}

func (r *Repository) List(ctx context.Context, params ListBeansParams) ([]models.Bean, int, error) {
	pag := models.NewPaginationParams(
		ptr.FromPtr(params.Page),
		ptr.FromPtr(params.Limit),
	)

	base := r.psql.
		Select(beanSelectColumns...).
		From(beansTable + " b").
		Join("users u ON u.id = b.author_id").
		Where("b.deleted_at IS NULL")

	if params.Query != nil && *params.Query != "" {
		like := "%" + *params.Query + "%"
		base = base.Where(sq.Or{
			sq.Like{"b.name": like},
			sq.Like{"b.roaster": like},
		})
	}

	var total int
	countBase := r.psql.
		Select("COUNT(*)").
		From(beansTable + " b").
		Where("b.deleted_at IS NULL")
	if params.Query != nil && *params.Query != "" {
		like := "%" + *params.Query + "%"
		countBase = countBase.Where(sq.Or{
			sq.Like{"b.name": like},
			sq.Like{"b.roaster": like},
		})
	}
	if err := countBase.QueryRowContext(ctx).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count beans: %w", err)
	}

	rows, err := base.
		OrderBy("b.created_at DESC").
		Limit(uint64(pag.GetLimit())).
		Offset(uint64(pag.Offset())).
		QueryContext(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list beans: %w", err)
	}
	defer rows.Close()

	var result []models.Bean
	for rows.Next() {
		bean, err := scanBean(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan bean: %w", err)
		}
		result = append(result, bean)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return result, total, nil
}

func (r *Repository) Update(ctx context.Context, beanID string, req models.BeanPayload) error {
	descriptorsJSON, err := marshalDescriptors(req.Descriptors)
	if err != nil {
		return err
	}
	_, err = r.psql.Update(beansTable).
		SetMap(map[string]any{
			"name":        req.Name,
			"roast_type":  req.RoastType,
			"roaster":     req.Roaster,
			"country":     req.Country,
			"region":      req.Region,
			"farm":        req.Farm,
			"process":     req.Process,
			"descriptors": descriptorsJSON,
			"q_score":     req.QScore,
			"url":         req.Url,
			"image_id":    req.ImageId,
		}).
		Where(sq.Eq{"id": beanID}).
		Where("deleted_at IS NULL").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update bean: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, beanID string) error {
	_, err := r.psql.Update(beansTable).
		Set("deleted_at", sq.Expr("datetime('now')")).
		Where(sq.Eq{"id": beanID}).
		Where("deleted_at IS NULL").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("soft delete bean: %w", err)
	}
	return nil
}

func scanBean(s scanner) (models.Bean, error) {
	var bean models.Bean
	var descriptorsJSON sql.NullString
	var country, region, farm, process, imageID, url sql.NullString
	var qScore sql.NullFloat64
	var authorAvatarID sql.NullString
	var roastType string

	err := s.Scan(
		&bean.Id, &bean.Name, &roastType, &bean.Roaster,
		&country, &region, &farm, &process,
		&descriptorsJSON, &qScore, &url, &imageID,
		&bean.CreatedAt,
		&bean.Author.Id, &bean.Author.Username, &authorAvatarID,
	)
	if err != nil {
		return models.Bean{}, err
	}

	bean.RoastType = models.BeanRoastType(roastType)

	if country.Valid {
		bean.Country = &country.String
	}
	if region.Valid {
		bean.Region = &region.String
	}
	if farm.Valid {
		bean.Farm = &farm.String
	}
	if process.Valid {
		bean.Process = &process.String
	}
	if imageID.Valid {
		bean.ImageId = &imageID.String
	}
	if url.Valid {
		bean.Url = &url.String
	}
	if qScore.Valid {
		v := float32(qScore.Float64)
		bean.QScore = &v
	}
	if authorAvatarID.Valid {
		bean.Author.AvatarId = &authorAvatarID.String
	}
	if descriptorsJSON.Valid && descriptorsJSON.String != "" {
		_ = json.Unmarshal([]byte(descriptorsJSON.String), &bean.Descriptors) // nolint:errcheck
	}

	return bean, nil
}

func marshalDescriptors(descriptors *[]string) (*string, error) {
	if descriptors == nil || len(*descriptors) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(*descriptors)
	if err != nil {
		return nil, fmt.Errorf("marshal descriptors: %w", err)
	}
	s := string(b)
	return &s, nil
}
