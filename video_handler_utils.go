package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	probeCmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	stdoutBuffer := bytes.Buffer{}
	probeCmd.Stdout = &stdoutBuffer

	err := probeCmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error at the ffprobe command runner: %w", err)
	}

	type Stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	type probeOutStr struct {
		Streams []Stream `json:"streams"`
	}

	probeOutPut := probeOutStr{}
	stdoutBytes := stdoutBuffer.Bytes()
	err = json.Unmarshal(stdoutBytes, &probeOutPut)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling buffer: %w", err)
	}

	tolerance := 0.03
	if len(probeOutPut.Streams) == 0 {
		return "", fmt.Errorf("Output has no stream")
	}

	firstStream := probeOutPut.Streams[0]
	ratio := float64(firstStream.Width) / float64(firstStream.Height)
	if math.Abs(ratio-16.0/9.0) < tolerance {
		return "16:9", nil
	} else if math.Abs(ratio-9.0/16.0) < tolerance {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	newFilePath := filePath + ".processing"
	command := exec.Command(
		"ffmpeg",
		"-i",
		filePath,
		"-c",
		"copy",
		"-movflags",
		"faststart",
		"-f",
		"mp4",
		newFilePath,
	)

	err := command.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffmpeg command: %w", err)
	}

	return newFilePath, nil
}

// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
// 	presignedClient := s3.NewPresignClient(s3Client)
// 	getObjInput := s3.GetObjectInput{
// 		Bucket: &bucket,
// 		Key:    &key,
// 	}
// 	opts := s3.WithPresignExpires(expireTime)
// 	req, err := presignedClient.PresignGetObject(context.Background(), &getObjInput, opts)
// 	if err != nil {
// 		return "", fmt.Errorf("error presign get object: %w", err)
// 	}
//
// 	return req.URL, nil
// }
//
// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
// 	if video.VideoURL == nil {
// 		return video, nil // no url is ok
// 	}
// 	bucketAndKey := strings.Split(*video.VideoURL, ",")
// 	bucket, key := bucketAndKey[0], bucketAndKey[1]
//
// 	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*10)
// 	if err != nil {
// 		const msg = "error generating presigned url in db to signed video: %w"
// 		return database.Video{}, fmt.Errorf(msg, err)
// 	}
//
// 	*video.VideoURL = presignedURL
// 	return video, nil
// }
