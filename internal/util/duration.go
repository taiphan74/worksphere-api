// Package util cung cấp các hàm tiện ích dùng chung trong toàn bộ ứng dụng.
package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration mở rộng time.ParseDuration với hỗ trợ suffix "d" cho ngày.
//
// Các định dạng được hỗ trợ:
//   - "7d" → 7 × 24h
//   - "1d12h" → 36h
//   - "1d12h30m" → 36h30m
//   - "" → 0
//
// Trả về error nếu chuỗi không hợp lệ hoặc có số ngày âm.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	dayStr, rest, hasDays := strings.Cut(s, "d")
	if hasDays {
		if dayStr == "" || dayStr == "-" || dayStr == "+" {
			return 0, fmt.Errorf("invalid duration %q", s)
		}

		days, err := strconv.Atoi(dayStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		if days < 0 {
			return 0, fmt.Errorf("negative days not allowed: %q", s)
		}

		dayDuration := time.Duration(days) * 24 * time.Hour
		if rest == "" {
			return dayDuration, nil
		}

		restDuration, err := time.ParseDuration(rest)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}

		return dayDuration + restDuration, nil
	}

	return time.ParseDuration(s)
}

// MustParseDuration gọi ParseDuration, panic nếu có lỗi.
func MustParseDuration(s string) time.Duration {
	d, err := ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
