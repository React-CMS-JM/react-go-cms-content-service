package repo

import (
	"context"
	"database/sql"

	"react-go-cms-content-service/internal/application/service/contenttype"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

// ContentTypeRepository reads content types from MySQL.
type ContentTypeRepository struct {
	db *sql.DB
}

// NewContentTypeRepository builds a content-type repository.
func NewContentTypeRepository(db *sql.DB) *ContentTypeRepository {
	return &ContentTypeRepository{db: db}
}

var _ contenttype.Repository = (*ContentTypeRepository)(nil)

// List returns content types ordered by id.
func (r *ContentTypeRepository) List(ctx context.Context) ([]entity.ContentType, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `SELECT id, name, slug, description FROM content_types ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.ContentType{}
	for rows.Next() {
		var row model.ContentTypeRow
		err = rows.Scan(&row.ID, &row.Name, &row.Slug, &row.Description)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.ContentType{
			ID:          row.ID,
			Name:        row.Name,
			Slug:        row.Slug,
			Description: StrPtr(row.Description),
		})
	}
	return out, rows.Err()
}
