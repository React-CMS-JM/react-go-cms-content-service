package entity

// ContentType is one content_types row.
type ContentType struct {
	ID          int
	Name        string
	Slug        string
	Description *string
}
