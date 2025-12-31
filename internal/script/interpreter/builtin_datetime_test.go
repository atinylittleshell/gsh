package interpreter

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestDateTimeNow(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	before := time.Now().UnixMilli()

	result, err := interp.EvalString(`DateTime.now()`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now().UnixMilli()

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	timestamp := int64(numVal.Value)
	if timestamp < before || timestamp > after {
		t.Errorf("expected timestamp between %d and %d, got %d", before, after, timestamp)
	}
}

func TestDateTimeNowNoArgs(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	_, err := interp.EvalString(`DateTime.now(123)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.now() with arguments")
	}
}

func TestDateTimeParseISO(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	result, err := interp.EvalString(`DateTime.parse("2024-01-15T10:30:00Z")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixMilli()
	if int64(numVal.Value) != expected {
		t.Errorf("expected %d, got %d", expected, int64(numVal.Value))
	}
}

func TestDateTimeParseDateOnly(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	result, err := interp.EvalString(`DateTime.parse("2024-01-15")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	if int64(numVal.Value) != expected {
		t.Errorf("expected %d, got %d", expected, int64(numVal.Value))
	}
}

func TestDateTimeParseWithFormat(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	result, err := interp.EvalString(`DateTime.parse("15/01/2024", "DD/MM/YYYY")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	if int64(numVal.Value) != expected {
		t.Errorf("expected %d, got %d", expected, int64(numVal.Value))
	}
}

func TestDateTimeParseInvalidString(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	_, err := interp.EvalString(`DateTime.parse("not a date")`, nil)
	if err == nil {
		t.Fatal("expected error when parsing invalid date string")
	}
}

func TestDateTimeFormatDefault(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Use a known timestamp (2024-01-15T10:30:00.000Z = 1705315800000 ms)
	timestamp := int64(1705315800000)

	result, err := interp.EvalString(`DateTime.format(1705315800000)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strVal, ok := result.FinalResult.(*StringValue)
	if !ok {
		t.Fatalf("expected string, got %s", result.FinalResult.Type())
	}

	// Default format is ISO 8601 in local time
	expected := time.UnixMilli(timestamp).Format("2006-01-02T15:04:05.000-07:00")
	if strVal.Value != expected {
		t.Errorf("expected %s, got %s", expected, strVal.Value)
	}
}

func TestDateTimeFormatCustom(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Use a timestamp in local time to avoid timezone issues
	// Create a local time and get its timestamp
	localTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	timestamp := localTime.UnixMilli()

	// Set the timestamp as a variable
	_, err := interp.EvalString(fmt.Sprintf(`ts = %d`, timestamp), nil)
	if err != nil {
		t.Fatalf("unexpected error setting timestamp: %v", err)
	}

	tests := []struct {
		format   string
		expected string
	}{
		{"YYYY-MM-DD", "2024-01-15"},
		{"DD/MM/YYYY", "15/01/2024"},
		{"HH:mm:ss", "10:30:00"},
		{"YYYY", "2024"},
		{"MMM DD, YYYY", "Jan 15, 2024"},
	}

	for _, tt := range tests {
		result, err := interp.EvalString(`DateTime.format(ts, "`+tt.format+`")`, nil)
		if err != nil {
			t.Fatalf("unexpected error for format %s: %v", tt.format, err)
		}

		strVal, ok := result.FinalResult.(*StringValue)
		if !ok {
			t.Fatalf("expected string, got %s", result.FinalResult.Type())
		}

		if strVal.Value != tt.expected {
			t.Errorf("format %s: expected %s, got %s", tt.format, tt.expected, strVal.Value)
		}
	}
}

func TestDateTimeDiffMilliseconds(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Two timestamps 1000ms apart
	result, err := interp.EvalString(`DateTime.diff(2000, 1000)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 1000 {
		t.Errorf("expected 1000, got %v", numVal.Value)
	}
}

func TestDateTimeDiffSeconds(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Two timestamps 5000ms apart
	result, err := interp.EvalString(`DateTime.diff(6000, 1000, "seconds")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 5 {
		t.Errorf("expected 5, got %v", numVal.Value)
	}
}

func TestDateTimeDiffMinutes(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Two timestamps 120000ms (2 minutes) apart
	result, err := interp.EvalString(`DateTime.diff(120000, 0, "minutes")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 2 {
		t.Errorf("expected 2, got %v", numVal.Value)
	}
}

func TestDateTimeDiffHours(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// 3600000ms = 1 hour
	result, err := interp.EvalString(`DateTime.diff(3600000, 0, "hours")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 1 {
		t.Errorf("expected 1, got %v", numVal.Value)
	}
}

func TestDateTimeDiffDays(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// 86400000ms = 1 day
	result, err := interp.EvalString(`DateTime.diff(86400000, 0, "days")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 1 {
		t.Errorf("expected 1, got %v", numVal.Value)
	}
}

func TestDateTimeDiffNegative(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// timestamp1 < timestamp2 should give negative result
	result, err := interp.EvalString(`DateTime.diff(1000, 2000)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != -1000 {
		t.Errorf("expected -1000, got %v", numVal.Value)
	}
}

func TestDateTimeDiffShortUnits(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Test short unit aliases
	tests := []struct {
		unit     string
		expected float64
	}{
		{"ms", 1000},
		{"s", 1},
		{"m", 1.0 / 60},
		{"h", 1.0 / 3600},
		{"d", 1.0 / 86400},
	}

	for _, tt := range tests {
		result, err := interp.EvalString(`DateTime.diff(1000, 0, "`+tt.unit+`")`, nil)
		if err != nil {
			t.Fatalf("unexpected error for unit %s: %v", tt.unit, err)
		}

		numVal, ok := result.FinalResult.(*NumberValue)
		if !ok {
			t.Fatalf("expected number, got %s", result.FinalResult.Type())
		}

		if math.Abs(numVal.Value-tt.expected) > 0.0001 {
			t.Errorf("unit %s: expected %v, got %v", tt.unit, tt.expected, numVal.Value)
		}
	}
}

func TestDateTimeDiffInvalidUnit(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	_, err := interp.EvalString(`DateTime.diff(1000, 0, "invalid")`, nil)
	if err == nil {
		t.Fatal("expected error for invalid unit")
	}
}

func TestDateTimeFormatTokens(t *testing.T) {
	// Test the format token conversion directly
	tests := []struct {
		dayjs    string
		expected string
	}{
		{"YYYY", "2006"},
		{"YY", "06"},
		{"MM", "01"},
		{"M", "1"},
		{"DD", "02"},
		{"D", "2"},
		{"HH", "15"},
		{"hh", "03"},
		{"mm", "04"},
		{"ss", "05"},
		{"SSS", "000"},
		{"A", "PM"},
		{"a", "pm"},
		{"Z", "-07:00"},
		{"ZZ", "-0700"},
		{"MMMM", "January"},
		{"MMM", "Jan"},
		{"dddd", "Monday"},
		{"ddd", "Mon"},
	}

	for _, tt := range tests {
		result := dayjsToGoFormat(tt.dayjs)
		if result != tt.expected {
			t.Errorf("dayjsToGoFormat(%s): expected %s, got %s", tt.dayjs, tt.expected, result)
		}
	}
}

func TestDateTimeRoundTrip(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Parse a date, format it back, parse again - should get same timestamp
	_, err := interp.EvalString(`
		ts1 = DateTime.parse("2024-06-15T14:30:00Z")
		formatted = DateTime.format(ts1, "YYYY-MM-DDTHH:mm:ssZ")
		ts2 = DateTime.parse(formatted)
		ts1 == ts2
	`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDateTimeIsBuiltin(t *testing.T) {
	if !isBuiltin("DateTime") {
		t.Error("expected DateTime to be registered as builtin")
	}
}

func TestDateTimeParseArgumentValidation(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// No arguments
	_, err := interp.EvalString(`DateTime.parse()`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.parse() with no arguments")
	}

	// Too many arguments
	_, err = interp.EvalString(`DateTime.parse("2024-01-01", "YYYY-MM-DD", "extra")`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.parse() with too many arguments")
	}

	// Wrong type for first argument
	_, err = interp.EvalString(`DateTime.parse(123)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.parse() with non-string first argument")
	}

	// Wrong type for second argument
	_, err = interp.EvalString(`DateTime.parse("2024-01-01", 123)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.parse() with non-string format")
	}
}

func TestDateTimeFormatArgumentValidation(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// No arguments
	_, err := interp.EvalString(`DateTime.format()`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.format() with no arguments")
	}

	// Too many arguments
	_, err = interp.EvalString(`DateTime.format(1000, "YYYY", "extra")`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.format() with too many arguments")
	}

	// Wrong type for first argument
	_, err = interp.EvalString(`DateTime.format("not a number")`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.format() with non-number timestamp")
	}

	// Wrong type for second argument
	_, err = interp.EvalString(`DateTime.format(1000, 123)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.format() with non-string format")
	}
}

func TestDateTimeDiffArgumentValidation(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Too few arguments
	_, err := interp.EvalString(`DateTime.diff(1000)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.diff() with too few arguments")
	}

	// Too many arguments
	_, err = interp.EvalString(`DateTime.diff(1000, 0, "ms", "extra")`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.diff() with too many arguments")
	}

	// Wrong type for first argument
	_, err = interp.EvalString(`DateTime.diff("not a number", 0)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.diff() with non-number first argument")
	}

	// Wrong type for second argument
	_, err = interp.EvalString(`DateTime.diff(1000, "not a number")`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.diff() with non-number second argument")
	}

	// Wrong type for third argument
	_, err = interp.EvalString(`DateTime.diff(1000, 0, 123)`, nil)
	if err == nil {
		t.Fatal("expected error when calling DateTime.diff() with non-string unit")
	}
}
