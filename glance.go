package main

import (
	"fmt"
	"math"

	"github.com/gregdel/pushover"
)

// buildGlance creates a Pushover Glance from supplied fields, retaining explicitly empty strings.
func buildGlance(args map[string]any, device string) (*pushover.Glance, error) {
	glance := &pushover.Glance{}

	if value, ok := args["title"]; ok {
		title, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("title must be a string")
		}
		glance.Title = pushover.String(title)
	}
	if value, ok := args["text"]; ok {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("text must be a string")
		}
		glance.Text = pushover.String(text)
	}
	if value, ok := args["subtext"]; ok {
		subtext, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("subtext must be a string")
		}
		glance.Subtext = pushover.String(subtext)
	}
	if value, ok := args["count"]; ok {
		count, err := glanceInteger(value, "count")
		if err != nil {
			return nil, err
		}
		glance.Count = pushover.Int(count)
	}
	if value, ok := args["percent"]; ok {
		percent, err := glanceInteger(value, "percent")
		if err != nil {
			return nil, err
		}
		if percent < 0 || percent > 100 {
			return nil, fmt.Errorf("percent must be between 0 and 100")
		}
		glance.Percent = pushover.Int(percent)
	}
	if device != "" {
		glance.DeviceName = device
	}
	if glance.Title == nil && glance.Text == nil && glance.Subtext == nil && glance.Count == nil && glance.Percent == nil {
		return nil, fmt.Errorf("at least one of title, text, subtext, count, or percent must be provided")
	}

	return glance, nil
}

// glanceInteger returns value as an int when it is an integral numeric value.
func glanceInteger(value any, name string) (int, error) {
	switch number := value.(type) {
	case int:
		return number, nil
	case int8:
		return int(number), nil
	case int16:
		return int(number), nil
	case int32:
		return int(number), nil
	case int64:
		if number < int64(math.MinInt) || number > int64(math.MaxInt) {
			return 0, fmt.Errorf("%s must fit in an int", name)
		}
		return int(number), nil
	case uint:
		if uint64(number) > uint64(math.MaxInt) {
			return 0, fmt.Errorf("%s must fit in an int", name)
		}
		return int(number), nil
	case uint8:
		return int(number), nil
	case uint16:
		return int(number), nil
	case uint32:
		if uint64(number) > uint64(math.MaxInt) {
			return 0, fmt.Errorf("%s must fit in an int", name)
		}
		return int(number), nil
	case uint64:
		if number > uint64(math.MaxInt) {
			return 0, fmt.Errorf("%s must fit in an int", name)
		}
		return int(number), nil
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
		if number < float64(math.MinInt) || number >= float64(math.MaxInt) {
			return 0, fmt.Errorf("%s must fit in an int", name)
		}
		return int(number), nil
	default:
		return 0, fmt.Errorf("%s must be an integer", name)
	}
}

// formatGlanceResult formats a successful Pushover Glance result for a caller.
func formatGlanceResult(resp *pushover.Response, actor string) string {
	quota := formatQuota(resp)
	if quota == "" {
		quota = "Quota: unknown"
	}
	result := "Glance sent. " + quota
	if actor != "" {
		result += " (sent by " + actor + ")"
	}
	return result
}
