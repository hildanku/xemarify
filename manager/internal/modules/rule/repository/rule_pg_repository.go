package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRuleRepository struct {
	db *pgxpool.Pool
}

// NewPgRuleRepository creates a Postgres-backed RuleRepository.
func NewPgRuleRepository(db *pgxpool.Pool) RuleRepository {
	return &pgRuleRepository{db: db}
}

func (r *pgRuleRepository) List(ctx context.Context, f ListFilter) ([]*domain.Rule, int, error) {
	allowedCols := map[string]string{
		"name":       "name",
		"level":      "level",
		"enabled":    "enabled",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortCol, ok := allowedCols[f.SortBy]
	if !ok {
		sortCol = "created_at"
	}

	direction := "DESC"
	if strings.EqualFold(string(f.Order), "asc") {
		direction = "ASC"
	}

	limit := 10
	if f.Limit > 0 {
		limit = f.Limit
	}
	offset := 0
	if f.Offset > 0 {
		offset = f.Offset
	}

	args := []any{}
	conditions := []string{}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := len(args)
		conditions = append(conditions,
			fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", n, n),
		)
	}
	if f.Level != "" {
		args = append(args, f.Level)
		conditions = append(conditions, fmt.Sprintf("level::text = $%d", len(args)))
	}
	if f.Enabled != nil {
		args = append(args, *f.Enabled)
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM rules %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	nLimit, nOffset := len(args)-1, len(args)
	dataQ := fmt.Sprintf(`
		SELECT id, name, description, level::text, enabled, condition, tags, version, created_by, created_at, updated_at
		FROM rules
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortCol, direction, nLimit, nOffset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rules []*domain.Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, 0, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

func (r *pgRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	const q = `
		SELECT id, name, description, level::text, enabled, condition, tags, version, created_by, created_at, updated_at
		FROM rules
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, q, id)
	rule, err := scanRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

func (r *pgRuleRepository) Create(ctx context.Context, rule *domain.Rule) error {
	condJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO rules (id, name, description, level, enabled, condition, tags, version, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4::severity, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err = r.db.Exec(ctx, q,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Level,
		rule.Enabled,
		condJSON,
		rule.Tags,
		rule.Version,
		rule.CreatedBy,
	)
	return err
}

func (r *pgRuleRepository) Update(ctx context.Context, rule *domain.Rule) error {
	condJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return err
	}

	const q = `
		UPDATE rules
		SET name        = $2,
		    description = $3,
		    level       = $4::severity,
		    enabled     = $5,
		    condition   = $6,
		    tags        = $7,
		    version     = version + 1,
		    updated_at  = NOW()
		WHERE id = $1
	`
	_, err = r.db.Exec(ctx, q,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Level,
		rule.Enabled,
		condJSON,
		rule.Tags,
	)
	return err
}

func (r *pgRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM rules WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// scanRule is a generic scanner that works with both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(row rowScanner) (*domain.Rule, error) {
	var rule domain.Rule
	var condJSON []byte
	var desc *string

	err := row.Scan(
		&rule.ID,
		&rule.Name,
		&desc,
		&rule.Level,
		&rule.Enabled,
		&condJSON,
		&rule.Tags,
		&rule.Version,
		&rule.CreatedBy,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if desc != nil {
		rule.Description = *desc
	}
	if rule.Tags == nil {
		rule.Tags = []string{}
	}
	if condJSON != nil {
		if err := json.Unmarshal(condJSON, &rule.Condition); err != nil {
			return nil, err
		}
	}
	return &rule, nil
}
