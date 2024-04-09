package parsing

import (
	"fmt"
	"strconv"
)

func ParseInt(val string) (int, error) {
	var result int
	var err error
	if result, err = strconv.Atoi(val); len(val) > 0 && err != nil {
		return 0, fmt.Errorf("error when parsing int value: %v", err)
	}
	return result, nil
}

func ParseInt64(val string) (int64, error) {
	var result int
	var err error
	if result, err = ParseInt(val); err != nil {
		return 0, err
	}
	return int64(result), nil
}

func ParseFloat64(val string) (float64, error) {
	var result float64
	var err error
	if result, err = strconv.ParseFloat(val, 64); len(val) > 0 && err != nil {
		return 0, fmt.Errorf("error when parsing float64 value: %v", err)
	}
	return result, nil
}
