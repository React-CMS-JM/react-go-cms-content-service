// Package entity holds content-service domain objects without transport tags.
package entity

// Page is an offset page of items.
type Page[T any] struct {
	Items []T
	Page  int
	Size  int
	Total int64
}
