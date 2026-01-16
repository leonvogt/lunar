package ui

import (
	"fmt"
	"time"
)

func FormatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(bytes)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%.1f PB", value)
}

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond * 100)

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func FormatAge(age time.Duration) string {
	if age == 0 {
		return "unknown"
	}

	days := int(age.Hours() / 24)
	hours := int(age.Hours()) % 24
	minutes := int(age.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	if hours > 0 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	if minutes > 0 {
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	return "just now"
}
