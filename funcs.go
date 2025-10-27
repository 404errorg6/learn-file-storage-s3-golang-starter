package main

import (
	"fmt"
	"strings"
)

func getType(str string) (string, error) {
	_, after, ok := strings.Cut(str, "/")
	if !ok {
		return "", fmt.Errorf("'/' not found in string")
	}
	return after, nil
}
