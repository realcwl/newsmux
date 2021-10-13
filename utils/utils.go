package utils

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/utils/dotenv"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/stat/distuv"
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

func AreJSONsEqual(s1, s2 string) (bool, error) {

	if len(s1) == 0 && len(s2) == 0 {
		// both invalid json, return true
		return true, nil
	} else if len(s1) == 0 || len(s2) == 0 {
		return false, nil
	}

	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, errors.Wrap(err, "Error mashalling string s1 in AreJSONsEqual()")
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, errors.Wrap(err, "Error mashalling string s2 in AreJSONsEqual()")
	}

	return reflect.DeepEqual(o1, o2), nil
}

func StringSlicesContainSameElements(s1, s2 []string) bool {
	sort.Strings(s1)
	sort.Strings(s2)
	return reflect.DeepEqual(s1, s2)
}

func StringifyBoolean(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func GetRandomDataCollectorFunctionName() string {
	return "data_collector_" + RandomAlphabetString(8)
}

func GetRandomNumberInRangeStandardDeviation(mean float64, radius float64) float64 {
	// Use 3 standard diviation, which has the 99.7% probability to succeed.
	deviation := float64(3)
	for {
		dist := distuv.UnitNormal
		num := dist.Rand()
		if num <= deviation && num >= -deviation {
			return num*radius/deviation + mean
		}
	}
}

func IsProdEnv() bool {
	return os.Getenv("NEWSMUX_ENV") == dotenv.ProdEnv
}
