package comment

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the comment persistence port.
type Repository interface {
	List(ctx context.Context, postID, status string) ([]entity.CommentRecord, error)
	PostExists(ctx context.Context, postID string) (bool, error)
	Insert(ctx context.Context, postID, userID string, parent *string, status, lang, content string) (string, error)
	Require(ctx context.Context, id string) error
	ReplaceContent(ctx context.Context, id, lang, content string) error
	ReadTranslation(ctx context.Context, id, lang string) (languageCode, content string, original bool, found bool, err error)
	FindTranslation(ctx context.Context, id, lang string) (rowID int, original bool, found bool, err error)
	InsertTranslation(ctx context.Context, id, lang, content string) error
	UpdateTranslation(ctx context.Context, rowID int, content string) error
	PatchStatus(ctx context.Context, id, status string) (bool, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (entity.CommentRecord, error)
}
