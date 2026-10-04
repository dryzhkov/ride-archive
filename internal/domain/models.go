// Package domain contains archive records independent of HTTP and storage.
package domain

import (
	"encoding/json"

	"ridearchive/internal/gpx"
)

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type CatalogItem struct {
	ID        string `json:"id"`
	OwnerID   string `json:"owner_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type Trip CatalogItem
type Bike CatalogItem

type EntryKind string

const (
	Unclassified   EntryKind = "unclassified"
	Recorded       EntryKind = "recorded"
	RouteReference EntryKind = "route_reference"
)

type ArchiveEntry struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Title     string    `json:"title"`
	Kind      EntryKind `json:"kind"`
	TripID    *string   `json:"trip_id"`
	BikeID    *string   `json:"bike_id"`
	Revision  int       `json:"revision"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type Source struct {
	OriginalFilename string `json:"original_filename"`
	ID               string `json:"id"`
	OwnerID          string `json:"owner_id"`
	SHA256           string `json:"sha256"`
	ByteLength       int    `json:"byte_length"`
	MediaType        string `json:"media_type"`
	CreatedAt        string `json:"created_at"`
}

// EntryChange preserves the accepted user assertion and its prior value.
type EntryChange struct {
	Revision   int             `json:"revision"`
	Origin     string          `json:"origin"`
	ActorID    string          `json:"actor_id"`
	Before     json.RawMessage `json:"before"`
	After      json.RawMessage `json:"after"`
	RecordedAt string          `json:"recorded_at"`
}

type EntryPatch struct {
	Revision int
	Title    *string
	TripSet  bool
	TripID   *string
	BikeSet  bool
	BikeID   *string
}

// PathSummary identifies an immutable parsed path; GPX structure is not a
// claim that the path was observed or ridden.
type PathSummary struct {
	ID               string `json:"id"`
	SourceID         string `json:"source_id"`
	OriginalFilename string `json:"original_filename"`
	Name             string `json:"name"`
	Structure        string `json:"structure"`
	SourceLocator    string `json:"source_locator"`
	PointCount       int    `json:"point_count"`
	SegmentCount     int    `json:"segment_count"`
}

type PathGeometry struct {
	PathSummary
	Segments [][]gpx.Point `json:"segments"`
}
