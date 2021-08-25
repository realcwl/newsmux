package utils

import (
	"math/rand"
	"strings"
	"time"
)

// ContainsString returns true iff the provided string slice hay contains string
// needle.
func ContainsString(hay []string, needle string) bool {
	for _, str := range hay {
		if str == needle {
			return true
		}
	}
	return false
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// This function will return random string of target length consisting
// alphabetic characters (lowercase) and number.
func RandomAlphabetString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

// for testing
func parseGQLTimeString(str string) (time.Time, error) {
	return time.Parse(time.RFC3339, str)
}

func serializeGQLTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
