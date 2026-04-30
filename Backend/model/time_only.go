package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TimeOnly stores a time-of-day without a date portion.
// Useful for representing clock times such as '09:30:00', regardless of date.
type TimeOnly struct {
	time.Time
}

// Layout for formatting and parsing time-of-day strings (24-hour format).
const (
	timeOnlyLayout      = "15:04:05" // e.g., "13:45:00"
	timeOnlyShortLayout = "15:04"    // e.g., "08:20"
)

// Scan implements the sql.Scanner interface for reading from SQL database columns of type TIME.
// Accepts time.Time, string, or []byte as input (commonly seen from database drivers).
// Stores the parsed time in TimeOnly. If the input is nil, sets to zero value (empty time).
func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parseString(string(v))
	case string:
		return t.parseString(v)
	default:
		return fmt.Errorf("unsupported scan type %T for TimeOnly", value)
	}
}

// Value implements the driver.Valuer interface for writing a TimeOnly value to a SQL database.
// Returns the time formatted as "HH:MM:SS", or nil if time value is zero.
func (t TimeOnly) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.Format(timeOnlyLayout), nil
}

// MarshalJSON serializes the TimeOnly value as a JSON string in the "HH:MM:SS" format.
// If value is zero, serializes as JSON null.
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte("\"" + t.Time.Format(timeOnlyLayout) + "\""), nil
}

// UnmarshalJSON deserializes a JSON string in "HH:MM:SS" or "HH:MM" format into a TimeOnly.
// If the input is JSON null, resets value to zero (empty time).
func (t *TimeOnly) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	trimmed := strings.Trim(string(data), "\"")
	return t.parseString(trimmed)
}

// parseString attempts to parse a string as a time-of-day in either "HH:MM:SS" or "HH:MM" format.
// On success, stores the parsed value in TimeOnly. If the string is empty, resets to zero time.
// Important: All parsing occurs in the server's local time zone context. Ensure all servers have the same TZ.
func (t *TimeOnly) parseString(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		t.Time = time.Time{}
		return nil
	}

	// NOTE: Course times are stored in server's LOCAL timezone context.
	// For correct operation, all servers MUST have the same TZ environment variable set.
	// Example for UTC-4: export TZ=America/New_York
	parsed, err := time.ParseInLocation(timeOnlyLayout, value, time.Local)
	if err != nil {
		parsed, err = time.ParseInLocation(timeOnlyShortLayout, value, time.Local)
		if err != nil {
			return errors.New("invalid time format for TimeOnly")
		}
	}
	t.Time = parsed
	return nil
}

// ParseTimeOnly is a helper function to parse a time-of-day string ("HH:MM" or "HH:MM:SS") and return a TimeOnly value.
// Returns an error if parsing fails.
func ParseTimeOnly(value string) (TimeOnly, error) {
	var t TimeOnly
	if err := t.parseString(value); err != nil {
		return TimeOnly{}, err
	}
	return t, nil
}