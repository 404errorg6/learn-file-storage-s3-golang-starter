package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	limit := int64(1 << 30)
	body := http.MaxBytesReader(w, r.Body, limit)
	r.Body = body
	videoIDStr := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid videoID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get token from http header", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not validate jwt", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not get video", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "You're not the video owner", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not read the form", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("content-type"))
	if err != nil || mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}
	vidType := getMediaType(mediaType)

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create tmp file", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy to tmp file", err)
		return
	}

	aspectRatio := "other"
	dimension, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid tmp path", err)
		return
	}

	if dimension == "16:9" {
		aspectRatio = "landscape"
	} else if dimension == "9:16" {
		aspectRatio = "portrait"
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not set pointer of tmp file to start", err)
		return
	}

	processedFileName, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not process video", err)
		return
	}

	processedFile, err := os.Open(processedFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not read the processed file", err)
		return
	}

	randomByte := make([]byte, 32)
	_, err = rand.Read(randomByte)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create file name", err)
		return
	}

	random := base64.URLEncoding.EncodeToString(randomByte)
	serverFileName := fmt.Sprintf("%v/%v.%v", aspectRatio, random, vidType)

	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &serverFileName,
		Body:        processedFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create object on server", err)
		return
	}

	vidURL := fmt.Sprintf("%v,%v", cfg.s3Bucket, serverFileName)
	video.VideoURL = &vidURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video", err)
		return
	}

	signedVid, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create signed URL", err)
		return
	}

	respondWithJSON(w, http.StatusOK, signedVid)
}
