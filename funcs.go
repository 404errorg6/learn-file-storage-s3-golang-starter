package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func getVideoAspectRatio(filePath string) (string, error) {
	fmt.Printf("%v\n", filePath)
	var dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	output := &bytes.Buffer{}
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(output.Bytes(), &dimensions)
	if err != nil {
		return "", err
	}

	ratio := float64(dimensions.Width / dimensions.Height)
	if ratio > 1.76 && ratio < 1.78 {
		return "16:9", nil
	} else if ratio > 0.55 && ratio < 0.57 {
		return "9:16", nil
	}
	return "other", nil
}

func getMediaType(str string) string {
	_, after, _ := strings.Cut(str, "/")
	return after
}
