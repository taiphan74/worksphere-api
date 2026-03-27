package mapper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TextPtr converts pgtype.Text to *string.
// Returns nil if the value is not valid (NULL in database).
func TextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

// TimestamptzPtr converts pgtype.Timestamptz to *time.Time.
// Returns nil if the value is not valid (NULL in database).
func TimestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

// StringToText converts *string to pgtype.Text.
// Returns pgtype.Text with Valid=false if the input is nil.
func StringToText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}

// TimeToTimestamptz converts *time.Time to pgtype.Timestamptz.
// Returns pgtype.Timestamptz with Valid=false if the input is nil.
func TimeToTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
