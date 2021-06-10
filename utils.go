package proxy

import (
	"fmt"
	"math/rand"
	"os"
)

/*
	This function does exactly what it says
*/
func GetEnvOrDefault(variable string, defaultValue string) string {
	if val, ok := os.LookupEnv(variable); ok {
		return val
	} else {
		return defaultValue
	}
}

func GetEnvOrPanic(variable string) string {
	if val, ok := os.LookupEnv(variable); ok {
		return val
	} else {
		panic(fmt.Errorf("unable to find env variable [%v]", variable))
	}
}

func IsVerbose() bool  {
	if _, ok := os.LookupEnv("VERBOSE"); ok {
		return true
	} else {
		return false
	}
}

func RandInRange(min int, max int) int {
	// shortcut for certain cases
	if max == min+1 {
		return min
	}
	return rand.Intn(max-min) + min
}
