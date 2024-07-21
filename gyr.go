// Gyr is sort of a standard library for my personal projects.
package gyr

import (
	"os"
	"regexp"
)

type SettingsFunc[SettingsStruct any] func(*SettingsStruct)

func isGyrDebug() bool {
	_, isSet := os.LookupEnv("GYR_DEBUG")
	return isSet
}

// Get the named matches from a regexp.
func regexNamedMatches(r *regexp.Regexp, str string) map[string]string {
	matchMap := make(map[string]string)
	match := r.FindStringSubmatch(str)
	for i, name := range r.SubexpNames() {
		if i != 0 {
			matchMap[name] = match[i]
		}
	}
	return matchMap
}
