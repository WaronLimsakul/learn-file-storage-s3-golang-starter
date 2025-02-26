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
	const maxByteMemory int64 = 1 << 30 // about 1GB
	// so basically this function just set a limit to my request body
	// Note that request.Body implements io.ReadCloser
	// io.ReadCloser are 2 interface combine = io.Reader + io.Closer
	// Reader = just for reading data (.Read())
	// Closer = for closing the resource when done (.Close())
	r.Body = http.MaxBytesReader(w, r.Body, maxByteMemory)

	videoID := r.PathValue("videoID")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse video id", err)
		return
	}

	jwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find jwt", err)
		return
	}

	userID, err := auth.ValidateJWT(jwt, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couludn't validate jwt", err)
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoUUID)
	if userID != videoMetaData.UserID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized", fmt.Errorf("Nope"))
		return
	}

	// we already set bytes limit, so we can be lazy and call
	// .FormFile() which will call .parseMultipartForm() for you
	videoFile, videoHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse video file", err)
		return
	}

	defer videoFile.Close()
	videoContentType := videoHeader.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(videoContentType)
	if mediaType != "video/mp4" {
		const msg = "Not a video/mp4 type"
		respondWithError(w, http.StatusBadRequest, msg, fmt.Errorf(msg))
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		const msg = "Couldn't create a temp file"
		respondWithError(w, http.StatusInternalServerError, msg, err)
		return
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close() // defer is LIFO, so close before remove

	_, err = io.Copy(tmpFile, videoFile)
	if err != nil {
		const msg = "Couldn't copy file content"
		respondWithError(w, http.StatusInternalServerError, msg, err)
		return
	}

	// .Seek() redirect tmpFile's file pointer to ...
	// 0 means the the start of ...
	// io.SeekStart means file starting point ...
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		const msg = "Could't redirect temp file pointer back"
		respondWithError(w, http.StatusInternalServerError, msg, err)
		return
	}

	var rand32Bytes [32]byte
	rand.Read(rand32Bytes[:])
	videoKey := base64.RawURLEncoding.EncodeToString(rand32Bytes[:])

	putObjectParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &videoKey,
		Body:        tmpFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't put a video in s3", err)
		return
	}

	newVideoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, videoKey)
	videoMetaData.VideoURL = &newVideoURL
	err = cfg.db.UpdateVideo(videoMetaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update vidoe url", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	return
}
