package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}

	thumbnailReqFile, reqFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}

	// DON'T media type from r.Header.Get(), it's gonna give you
	// multi-part form instead of those "image/png", "image/jpg"
	// you must get it from header of the parsed file
	mediaType := reqFileHeader.Header.Get("Content-Type")

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video metadata", err)
		return
	}

	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of video", err)
		return
	}

	thumbnailExtension, ok := strings.CutPrefix(mediaType, "image/")
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "Couldn't cut a media type prefix", err)
		return
	}

	thumbnailPath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoMetaData.ID, thumbnailExtension))
	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file in disk", err)
		return
	}

	// thumbnailFile is io.Writer in the file system
	// thumbnailReqFile is the io.Reader file in the request
	_, err = io.Copy(thumbnailFile, thumbnailReqFile)
	if err != nil {
		log.Printf("mediaType: %s\n", mediaType)
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file to disk", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID, mediaType)
	videoMetaData.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video's thumbnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetaData)
}
