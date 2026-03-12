package repository

import surrealdb "github.com/surrealdb/surrealdb.go"

// flattenQueryResults collapses a SurrealDB Query result into a flat slice.
//
// Query[[]T] returns *[]QueryResult[[]T], where each element represents the
// results of one SQL statement. For single-statement queries this unwraps
// the inner []T from the first result set.
func flattenQueryResults[T any](results *[]surrealdb.QueryResult[[]T]) []T {
	if results == nil {
		return nil
	}
	var out []T
	for _, r := range *results {
		out = append(out, r.Result...)
	}
	return out
}
