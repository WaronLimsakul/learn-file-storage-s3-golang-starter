package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
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

	// anything that can be opend (e.g. file), must be close
	defer thumbnailReqFile.Close()

	// DON'T get media type from r.Header.Get(), it's gonna give you
	// "multi-part form" instead of those "image/png", "image/jpg"
	// you must get it from header of the parsed file
	fileContentType := reqFileHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileContentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse media type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		const msg = "Only jpg and png accepted"
		respondWithError(w, http.StatusBadRequest, msg, fmt.Errorf("media type: %s", mediaType))
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video metadata", fmt.Errorf(""))
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

	// a chunk of 32 bytes, no pointer
	var rand32Bytes [32]byte
	// create a slice pointing to the array
	rand.Read(rand32Bytes[:])
	// encode it to use in url
	thumbnailName := base64.RawURLEncoding.EncodeToString(rand32Bytes[:])

	thumbnailPath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", thumbnailName, thumbnailExtension))
	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		fmt.Printf("err: %s\n", err)
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

	// don't forget : media type = image/png, but extension = png
	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, thumbnailPath)
	videoMetaData.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video's thumbnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetaData)
}
