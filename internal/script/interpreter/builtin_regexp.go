package interpreter

import (
	"fmt"
	"regexp"
	"sync"
)

// regexpCache is a thread-safe LRU cache for compiled regular expressions.
// This avoids repeated compilation of the same pattern, which is expensive.
type regexpCache struct {
	mu      sync.RWMutex
	cache   map[string]*regexp.Regexp
	order   []string // tracks insertion order for LRU eviction
	maxSize int
}

// Global regex cache with a reasonable size limit
var globalRegexpCache = &regexpCache{
	cache:   make(map[string]*regexp.Regexp),
	order:   make([]string, 0, 128),
	maxSize: 128, // Cache up to 128 compiled patterns
}

// getOrCompile returns a cached compiled regex or compiles and caches a new one.
// Returns the compiled regex and any compilation error.
func (c *regexpCache) getOrCompile(pattern string) (*regexp.Regexp, error) {
	// Fast path: check if already cached (read lock)
	c.mu.RLock()
	if re, ok := c.cache[pattern]; ok {
		c.mu.RUnlock()
		return re, nil
	}
	c.mu.RUnlock()

	// Slow path: compile and cache (write lock)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if re, ok := c.cache[pattern]; ok {
		return re, nil
	}

	// Compile the pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	// Evict oldest entry if at capacity
	if len(c.cache) >= c.maxSize {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.cache, oldest)
	}

	// Add to cache
	c.cache[pattern] = re
	c.order = append(c.order, pattern)

	return re, nil
}

// compileRegexp compiles a regex pattern using the global cache
func compileRegexp(pattern string) (*regexp.Regexp, error) {
	return globalRegexpCache.getOrCompile(pattern)
}

// createRegexpObject creates the Regexp object with static methods for regular expression operations
// - Regexp.match(str, pattern) - returns an array of matches or null if no match
// - Regexp.test(str, pattern) - returns true if pattern matches the string
// - Regexp.replace(str, pattern, replacement) - replaces matches with replacement string
// - Regexp.replaceAll(str, pattern, replacement) - replaces all matches with replacement string
// - Regexp.split(str, pattern) - splits string by pattern
// - Regexp.findAll(str, pattern) - returns all matches as an array
// - Regexp.escape(str) - escapes special regex characters in a string
func createRegexpObject() *ObjectValue {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"match": {Value: &BuiltinValue{
				Name: "Regexp.match",
				Fn:   builtinRegexpMatch,
			}, ReadOnly: true},
			"test": {Value: &BuiltinValue{
				Name: "Regexp.test",
				Fn:   builtinRegexpTest,
			}, ReadOnly: true},
			"replace": {Value: &BuiltinValue{
				Name: "Regexp.replace",
				Fn:   builtinRegexpReplace,
			}, ReadOnly: true},
			"replaceAll": {Value: &BuiltinValue{
				Name: "Regexp.replaceAll",
				Fn:   builtinRegexpReplaceAll,
			}, ReadOnly: true},
			"split": {Value: &BuiltinValue{
				Name: "Regexp.split",
				Fn:   builtinRegexpSplit,
			}, ReadOnly: true},
			"findAll": {Value: &BuiltinValue{
				Name: "Regexp.findAll",
				Fn:   builtinRegexpFindAll,
			}, ReadOnly: true},
			"escape": {Value: &BuiltinValue{
				Name: "Regexp.escape",
				Fn:   builtinRegexpEscape,
			}, ReadOnly: true},
		},
	}
}

// builtinRegexpMatch implements Regexp.match(str, pattern)
// Returns an array with the match and any capture groups, or null if no match
// The first element is the full match, subsequent elements are capture groups
func builtinRegexpMatch(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Regexp.match() takes exactly 2 arguments (str, pattern), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.match() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.match() second argument must be a string (pattern), got %s", args[1].Type())
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.match() invalid pattern '%s': %v", patternVal.Value, err)
	}

	matches := re.FindStringSubmatch(strVal.Value)
	if matches == nil {
		return &NullValue{}, nil
	}

	// Convert matches to array
	elements := make([]Value, len(matches))
	for i, match := range matches {
		elements[i] = &StringValue{Value: match}
	}

	return &ArrayValue{Elements: elements}, nil
}

// builtinRegexpTest implements Regexp.test(str, pattern)
// Returns true if the pattern matches the string, false otherwise
func builtinRegexpTest(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Regexp.test() takes exactly 2 arguments (str, pattern), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.test() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.test() second argument must be a string (pattern), got %s", args[1].Type())
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.test() invalid pattern '%s': %v", patternVal.Value, err)
	}

	return &BoolValue{Value: re.MatchString(strVal.Value)}, nil
}

// builtinRegexpReplace implements Regexp.replace(str, pattern, replacement)
// Replaces the first match of pattern in str with replacement
// Replacement can include $1, $2, etc. for capture group references
func builtinRegexpReplace(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("Regexp.replace() takes exactly 3 arguments (str, pattern, replacement), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replace() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replace() second argument must be a string (pattern), got %s", args[1].Type())
	}

	replacementVal, ok := args[2].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replace() third argument must be a string (replacement), got %s", args[2].Type())
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.replace() invalid pattern '%s': %v", patternVal.Value, err)
	}

	// Replace only the first match
	result := replaceFirst(re, strVal.Value, replacementVal.Value)
	return &StringValue{Value: result}, nil
}

// replaceFirst replaces only the first match of the pattern
func replaceFirst(re *regexp.Regexp, str, replacement string) string {
	loc := re.FindStringIndex(str)
	if loc == nil {
		return str
	}
	// Use ReplaceAllString on just the matched portion to handle group references
	matched := str[loc[0]:loc[1]]
	replaced := re.ReplaceAllString(matched, replacement)
	return str[:loc[0]] + replaced + str[loc[1]:]
}

// builtinRegexpReplaceAll implements Regexp.replaceAll(str, pattern, replacement)
// Replaces all matches of pattern in str with replacement
// Replacement can include $1, $2, etc. for capture group references
func builtinRegexpReplaceAll(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("Regexp.replaceAll() takes exactly 3 arguments (str, pattern, replacement), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replaceAll() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replaceAll() second argument must be a string (pattern), got %s", args[1].Type())
	}

	replacementVal, ok := args[2].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.replaceAll() third argument must be a string (replacement), got %s", args[2].Type())
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.replaceAll() invalid pattern '%s': %v", patternVal.Value, err)
	}

	result := re.ReplaceAllString(strVal.Value, replacementVal.Value)
	return &StringValue{Value: result}, nil
}

// builtinRegexpSplit implements Regexp.split(str, pattern)
// Splits the string by the pattern and returns an array of substrings
func builtinRegexpSplit(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("Regexp.split() takes 2 or 3 arguments (str, pattern, limit?), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.split() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.split() second argument must be a string (pattern), got %s", args[1].Type())
	}

	limit := -1 // No limit by default
	if len(args) == 3 {
		limitVal, ok := args[2].(*NumberValue)
		if !ok {
			return nil, fmt.Errorf("Regexp.split() third argument must be a number (limit), got %s", args[2].Type())
		}
		limit = int(limitVal.Value)
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.split() invalid pattern '%s': %v", patternVal.Value, err)
	}

	parts := re.Split(strVal.Value, limit)
	elements := make([]Value, len(parts))
	for i, part := range parts {
		elements[i] = &StringValue{Value: part}
	}

	return &ArrayValue{Elements: elements}, nil
}

// builtinRegexpFindAll implements Regexp.findAll(str, pattern)
// Returns an array of all matches (without capture groups)
func builtinRegexpFindAll(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("Regexp.findAll() takes 2 or 3 arguments (str, pattern, limit?), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.findAll() first argument must be a string, got %s", args[0].Type())
	}

	patternVal, ok := args[1].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.findAll() second argument must be a string (pattern), got %s", args[1].Type())
	}

	limit := -1 // No limit by default (find all)
	if len(args) == 3 {
		limitVal, ok := args[2].(*NumberValue)
		if !ok {
			return nil, fmt.Errorf("Regexp.findAll() third argument must be a number (limit), got %s", args[2].Type())
		}
		limit = int(limitVal.Value)
	}

	re, err := compileRegexp(patternVal.Value)
	if err != nil {
		return nil, fmt.Errorf("Regexp.findAll() invalid pattern '%s': %v", patternVal.Value, err)
	}

	matches := re.FindAllString(strVal.Value, limit)
	if matches == nil {
		return &ArrayValue{Elements: []Value{}}, nil
	}

	elements := make([]Value, len(matches))
	for i, match := range matches {
		elements[i] = &StringValue{Value: match}
	}

	return &ArrayValue{Elements: elements}, nil
}

// builtinRegexpEscape implements Regexp.escape(str)
// Escapes all special regex metacharacters in the string
func builtinRegexpEscape(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Regexp.escape() takes exactly 1 argument (str), got %d", len(args))
	}

	strVal, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("Regexp.escape() argument must be a string, got %s", args[0].Type())
	}

	escaped := regexp.QuoteMeta(strVal.Value)
	return &StringValue{Value: escaped}, nil
}
