package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func genenratePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	singedClient := s3.NewPresignClient(s3Client)
	params := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	request, err := singedClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	newFile := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return newFile, nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	var probeOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	output := &bytes.Buffer{}
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(output.Bytes(), &probeOutput)
	if err != nil {
		return "", err
	}

	dimensions := probeOutput.Streams[0]

	if dimensions.Height == 0 {
		dimensions.Height = 1
	}
	if dimensions.Width == 0 {
		dimensions.Width = 1
	}

	ratio := float64(dimensions.Width) / float64(dimensions.Height)
	if ratio > 1.76 && ratio < 1.78 {
		return "16:9", nil
	} else if ratio > 0.54 && ratio < 0.58 {
		return "9:16", nil
	}
	return "other", nil
}

func getMediaType(str string) string {
	_, after, _ := strings.Cut(str, "/")
	return after
}
