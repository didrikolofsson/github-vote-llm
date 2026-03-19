package helpers

import "github.com/jackc/pgx/v5/pgtype"

// float64ToNumeric converts *float64 to pgtype.Numeric.
func Float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*f); err != nil {
		return pgtype.Numeric{}
	}
	return n
}
