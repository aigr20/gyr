package gyr

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

// The file [LoadEnvironment] will attempt to read environment variables from. Default is '.env'.
var EnvFile = ".env"
var lineMatcher = regexp.MustCompile(`^(?P<name>[a-zA-Z][a-zA-Z0-9_]+)=(?P<value>\S+)$`)

// Reads variables in the file specified by [EnvFile] into the current environment.
func LoadEnvironment() error {
	file, err := os.Open(EnvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		line = strings.TrimSpace(line)
		if shouldSkipLine(line) {
			continue
		}
		matches := regexNamedMatches(lineMatcher, line)
		if _, isSet := os.LookupEnv(matches["name"]); isSet || len(matches) != 2 {
			continue
		}
		os.Setenv(matches["name"], matches["value"])
	}
	return nil
}

func shouldSkipLine(line string) bool {
	if strings.HasPrefix(line, "#") || len(line) == 0 || !lineMatcher.MatchString(line) {
		return true
	}
	return false
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
