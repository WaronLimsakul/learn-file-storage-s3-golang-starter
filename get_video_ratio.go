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
