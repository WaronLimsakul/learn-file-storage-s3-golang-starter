package main

import (
	"fmt"
	"os/exec"
)

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
