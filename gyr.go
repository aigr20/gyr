package gyr

import "os"

type SettingsFunc[SettingsStruct any] func(*SettingsStruct)

func isGyrDebug() bool {
	_, isSet := os.LookupEnv("GYR_DEBUG")
	return isSet
}
