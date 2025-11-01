package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	bucket, key, found := strings.Cut(*video.VideoURL, ",")
	if !found {
		fmt.Printf("VideoURL: %v\n", *video.VideoURL)
		return video, fmt.Errorf("Could not find ',' in Video URL")
	}

	signedURL, err := genenratePresignedURL(cfg.s3Client, bucket, key, 1*time.Hour)
	if err != nil {
		return video, err
	}
	video.VideoURL = &signedURL
	return video, nil
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0o755)
	}
	return nil
}
