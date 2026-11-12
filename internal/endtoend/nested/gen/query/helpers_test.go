package query

import "github.com/jackc/pgx/v5/pgtype"

// id builds a deterministic valid pgtype.UUID from one byte.
func id(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Bytes[15] = b
	u.Valid = true
	return u
}
