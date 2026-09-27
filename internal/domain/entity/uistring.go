package entity

import "time"

// UIString is one localized UI string.
type UIString struct {
	StringKey    string
	UIComponent  *string
	LanguageCode string
	StringValue  string
	UpdatedAt    *time.Time
}

// UIStringWrite is one UI-string upsert.
type UIStringWrite struct {
	StringKey    string
	LanguageCode string
	StringValue  string
}
