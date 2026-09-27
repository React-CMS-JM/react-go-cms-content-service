package entity

import "time"

// TypeCounts is the dashboard tally of owned content types.
type TypeCounts struct {
	Post    int64
	Page    int64
	Service int64
	Product int64
}

// RecentPost is one row on the dashboard recent list, before the author name is attached.
type RecentPost struct {
	ID            string
	AuthorID      string
	ContentTypeID int
	Status        string
	UpdatedAt     *time.Time
}

// Recent is a dashboard recent post with its display fields.
type Recent struct {
	ID              string
	Title           string
	AuthorID        string
	AuthorName      string
	Status          string
	ContentTypeSlug string
	UpdatedAt       *time.Time
}

// Dashboard is the admin dashboard payload.
type Dashboard struct {
	Counts          TypeCounts
	PendingComments int64
	Recent          []Recent
}
