package ytapi

// Quota costs for YouTube Data API v3 operations.
// Reference: https://developers.google.com/youtube/v3/determine_quota_cost
//
// These constants are the single source of truth for cost numbers used in
// --help text and --dry-run output across the CLI. If Google changes a price,
// update it here and every command picks it up.
const (
	// Reads: 1 unit per call regardless of pageSize.
	CostList = 1

	// Writes: 50 units each.
	CostInsert = 50
	CostUpdate = 50
	CostDelete = 50
	CostRate   = 50

	// Cheap reads.
	CostGetRating = 1

	// Expensive reads.
	CostSearchList = 100

	// Uploads.
	CostVideoUpload = 1600
)
