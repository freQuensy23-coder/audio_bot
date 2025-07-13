package media

import (
	"context"
	"fmt"
	"os/exec"
)

// ExtractAudio executes ffmpeg to extract an audio track.
func ExtractAudio(ctx context.Context, ffmpegPath, inputPath string) (string, error) {
	outputPath := inputPath + ".mp3"
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", inputPath,
		"-vn", // Disable video
		"-acodec", "libmp3lame",
		"-q:a", "2", // VBR quality
		outputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	return outputPath, nil
}
