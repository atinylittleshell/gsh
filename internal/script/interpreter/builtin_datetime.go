package interpreter

import (
	"fmt"
	"time"
)

// createDateTimeObject creates the DateTime object with static methods
// Similar to dayjs but with static-only API:
// - DateTime.now() - returns current timestamp in milliseconds
// - DateTime.parse(str, format?) - parses a date string into timestamp
// - DateTime.format(timestamp, format?) - formats a timestamp into string
// - DateTime.diff(timestamp1, timestamp2, unit?) - returns difference between two timestamps
func createDateTimeObject() *ObjectValue {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"now": {Value: &BuiltinValue{
				Name: "DateTime.now",
				Fn:   builtinDateTimeNow,
			}, ReadOnly: true},
			"parse": {Value: &BuiltinValue{
				Name: "DateTime.parse",
				Fn:   builtinDateTimeParse,
			}, ReadOnly: true},
			"format": {Value: &BuiltinValue{
				Name: "DateTime.format",
				Fn:   builtinDateTimeFormat,
			}, ReadOnly: true},
			"diff": {Value: &BuiltinValue{
				Name: "DateTime.diff",
				Fn:   builtinDateTimeDiff,
			}, ReadOnly: true},
		},
	}
}

// builtinDateTimeNow implements DateTime.now()
// Returns the current timestamp in milliseconds since Unix epoch
func builtinDateTimeNow(args []Value) (Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("DateTime.now() takes no arguments, got %d", len(args))
	}
	return &NumberValue{Value: float64(time.Now().UnixMilli())}, nil
}

// builtinDateTimeParse implements DateTime.parse(str, format?)
// Parses a date string and returns timestamp in milliseconds
// If format is not provided, tries common formats
func builtinDateTimeParse(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("DateTime.parse() takes 1 or 2 arguments (str, format?), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("DateTime.parse() first argument must be a string, got %s", args[0].Type())
	}

	var t time.Time
	var err error

	if len(args) == 2 {
		// Custom format provided
		formatVal, ok := args[1].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("DateTime.parse() second argument must be a string (format), got %s", args[1].Type())
		}
		goFormat := dayjsToGoFormat(formatVal.Value)
		t, err = time.Parse(goFormat, strVal.Value)
		if err != nil {
			return nil, fmt.Errorf("DateTime.parse() failed to parse '%s' with format '%s': %v", strVal.Value, formatVal.Value, err)
		}
	} else {
		// Try common formats
		t, err = parseWithCommonFormats(strVal.Value)
		if err != nil {
			return nil, fmt.Errorf("DateTime.parse() failed to parse '%s': %v", strVal.Value, err)
		}
	}

	return &NumberValue{Value: float64(t.UnixMilli())}, nil
}

// builtinDateTimeFormat implements DateTime.format(timestamp, format?)
// Formats a timestamp (in milliseconds) into a string
// Default format is ISO 8601 (YYYY-MM-DDTHH:mm:ss.SSSZ)
func builtinDateTimeFormat(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("DateTime.format() takes 1 or 2 arguments (timestamp, format?), got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("DateTime.format() first argument must be a number (timestamp in ms), got %s", args[0].Type())
	}

	t := time.UnixMilli(int64(numVal.Value))

	format := "YYYY-MM-DDTHH:mm:ss.SSSZ" // ISO 8601 default
	if len(args) == 2 {
		formatVal, ok := args[1].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("DateTime.format() second argument must be a string (format), got %s", args[1].Type())
		}
		format = formatVal.Value
	}

	goFormat := dayjsToGoFormat(format)
	result := t.Format(goFormat)

	return &StringValue{Value: result}, nil
}

// builtinDateTimeDiff implements DateTime.diff(timestamp1, timestamp2, unit?)
// Returns the difference between two timestamps
// Unit can be: "milliseconds" (default), "seconds", "minutes", "hours", "days"
func builtinDateTimeDiff(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("DateTime.diff() takes 2 or 3 arguments (timestamp1, timestamp2, unit?), got %d", len(args))
	}

	ts1, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("DateTime.diff() first argument must be a number (timestamp in ms), got %s", args[0].Type())
	}

	ts2, ok := args[1].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("DateTime.diff() second argument must be a number (timestamp in ms), got %s", args[1].Type())
	}

	unit := "milliseconds"
	if len(args) == 3 {
		unitVal, ok := args[2].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("DateTime.diff() third argument must be a string (unit), got %s", args[2].Type())
		}
		unit = unitVal.Value
	}

	diffMs := ts1.Value - ts2.Value

	var result float64
	switch unit {
	case "milliseconds", "ms":
		result = diffMs
	case "seconds", "s":
		result = diffMs / 1000
	case "minutes", "m":
		result = diffMs / (1000 * 60)
	case "hours", "h":
		result = diffMs / (1000 * 60 * 60)
	case "days", "d":
		result = diffMs / (1000 * 60 * 60 * 24)
	default:
		return nil, fmt.Errorf("DateTime.diff() unknown unit '%s', expected: milliseconds, seconds, minutes, hours, days", unit)
	}

	return &NumberValue{Value: result}, nil
}

// dayjsToGoFormat converts dayjs-style format tokens to Go time format
// Common tokens:
// YYYY -> 2006, YY -> 06
// MM -> 01, M -> 1
// DD -> 02, D -> 2
// HH -> 15 (24h), hh -> 03 (12h), h -> 3
// mm -> 04, m -> 4
// ss -> 05, s -> 5
// SSS -> .000 (milliseconds)
// A -> PM, a -> pm
// Z -> -07:00, ZZ -> -0700
func dayjsToGoFormat(format string) string {
	// Token mappings - longer tokens first to avoid partial matches
	tokens := []struct {
		from string
		to   string
	}{
		{"YYYY", "2006"},
		{"YY", "06"},
		{"MMMM", "January"},
		{"MMM", "Jan"},
		{"MM", "01"},
		{"DDDD", "002"}, // Day of year
		{"DD", "02"},
		{"dddd", "Monday"},
		{"ddd", "Mon"},
		{"HH", "15"},
		{"hh", "03"},
		{"mm", "04"},
		{"ss", "05"},
		{"SSS", "000"},
		{"ZZ", "-0700"},
		{"A", "PM"},
		{"a", "pm"},
		// Single char tokens last (most ambiguous)
		{"Z", "-07:00"},
		{"M", "1"},
		{"D", "2"},
		{"H", "15"},
		{"h", "3"},
		{"m", "4"},
		{"s", "5"},
	}

	var result []byte
	i := 0
	for i < len(format) {
		matched := false
		// Try to match tokens (longer ones first due to ordering)
		for _, t := range tokens {
			if i+len(t.from) <= len(format) && format[i:i+len(t.from)] == t.from {
				result = append(result, t.to...)
				i += len(t.from)
				matched = true
				break
			}
		}
		if !matched {
			result = append(result, format[i])
			i++
		}
	}
	return string(result)
}

// parseWithCommonFormats tries to parse a date string with common formats
func parseWithCommonFormats(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date string with any known format")
}
