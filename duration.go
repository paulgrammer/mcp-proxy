package proxy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Duration is a wrapper around time.Duration that marshals to seconds
type Duration time.Duration

// String returns the duration as a string in standard Go duration format
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON converts the duration to a string in seconds like "60s"
func (d Duration) MarshalJSON() ([]byte, error) {
	seconds := time.Duration(d).Seconds()
	// Format with no decimal places if it's a whole number
	if seconds == float64(int64(seconds)) {
		return json.Marshal(fmt.Sprintf("%.0fs", seconds))
	}
	return json.Marshal(fmt.Sprintf("%gs", seconds))
}

// UnmarshalJSON converts JSON data to duration supporting multiple formats:
// - Numbers (30) as seconds
// - Numeric strings ("30") as seconds
// - Duration strings ("30s", "1m30s", etc.)
func (d *Duration) UnmarshalJSON(data []byte) error {
	var rawValue interface{}
	if err := json.Unmarshal(data, &rawValue); err != nil {
		return err
	}

	return d.parseValue(rawValue)
}

// MarshalYAML converts the duration to a string in seconds like "60s" for YAML
func (d Duration) MarshalYAML() (interface{}, error) {
	seconds := time.Duration(d).Seconds()
	// Format with no decimal places if it's a whole number
	if seconds == float64(int64(seconds)) {
		return fmt.Sprintf("%.0fs", seconds), nil
	}
	return fmt.Sprintf("%gs", seconds), nil
}

// UnmarshalYAML converts YAML data to duration supporting multiple formats:
// - Numbers (30) as seconds
// - Numeric strings ("30") as seconds
// - Duration strings ("30s", "1m30s", etc.)
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawValue interface{}
	if err := unmarshal(&rawValue); err != nil {
		return err
	}

	return d.parseValue(rawValue)
}

// parseValue is a shared helper for parsing duration values from JSON/YAML
func (d *Duration) parseValue(rawValue interface{}) error {
	switch v := rawValue.(type) {
	case float64:
		// Direct number (30)
		*d = Duration(time.Duration(v * float64(time.Second)))
		return nil

	case int:
		// Integer value
		*d = Duration(time.Duration(v) * time.Second)
		return nil

	case string:
		// Try parsing as duration string first ("30s", "1m", etc.)
		if parsed, err := time.ParseDuration(v); err == nil {
			*d = Duration(parsed)
			return nil
		}

		// Try parsing as numeric string ("30")
		if seconds, err := strconv.ParseFloat(v, 64); err == nil {
			*d = Duration(time.Duration(seconds * float64(time.Second)))
			return nil
		}

		return fmt.Errorf("invalid duration format: %q", v)

	default:
		return fmt.Errorf("duration must be a number or string, got %T", rawValue)
	}
}
