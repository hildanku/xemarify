package query

// SortOrder controls the direction of an ORDER BY clause.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// BaseFilter is a reusable embed for list queries that need
// full-text search, sorting, and offset-based pagination.
// Embed this in any module-specific ListFilter to get the shared contract for free.
//
// Example:
//
//	type ListFilter struct {
//	    query.BaseFilter          // shared fields
//	    Status string             // module-specific field
//	}
type BaseFilter struct {
	// Search performs a case-insensitive partial match; exact columns are
	// decided by each repository implementation.
	Search string

	// SortBy is the column name to sort results by.
	// Each repository maps this to a whitelisted column; unknown values fall
	// back to the repository's default (usually created_at).
	SortBy string

	// Order is the sort direction. Defaults to ASC when empty or unrecognised.
	Order SortOrder

	// Limit is the maximum number of rows to return. Defaults to 10.
	Limit int

	// Offset is the number of rows to skip before returning results.
	Offset int
}
