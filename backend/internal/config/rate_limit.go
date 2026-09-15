package config

import "strconv"
import "os"

func RateLimitPerMinute() int {
	if n, e := strconv.Atoi(os.Getenv("RATE_LIMIT_PER_MINUTE")); e == nil && n > 0 {
		return n
	}
	return 120
}
