package audio

import (
	"fmt"
	"math"
	"os"

	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
)

var decryptLog = logging.Component("decrypt")

// DecryptAAX decrypts an AAX file using activation bytes.
// Output is an M4B file (same codec, just container copy).
func (f *FFmpeg) DecryptAAX(inputPath, outputPath, activationBytes string, progressCb func(ProgressInfo)) error {
	return f.DecryptAAXWithMetadata(inputPath, outputPath, activationBytes, Metadata{}, progressCb)
}

// DecryptAAXWithMetadata decrypts AAX and embeds metadata in one ffmpeg invocation.
func (f *FFmpeg) DecryptAAXWithMetadata(inputPath, outputPath, activationBytes string, meta Metadata, progressCb func(ProgressInfo)) error {
	decryptLog.Info().
		Str("input", inputPath).
		Str("output", outputPath).
		Msg("decrypting AAX")

	err := f.runWithProgress(f.buildDecryptArgs(inputPath, outputPath, activationBytes, "", "", meta), progressCb)
	if err != nil {
		decryptLog.Error().Err(err).Str("input", inputPath).Msg("AAX decryption failed")
		return fmt.Errorf("AAX decryption failed: %w", err)
	}

	decryptLog.Debug().Str("output", outputPath).Msg("AAX decryption succeeded, validating")
	return f.validateDecryption(inputPath, outputPath, activationBytes)
}

// DecryptAAXC decrypts an AAXC file using key and IV.
// Output is an M4B file (same codec, just container copy).
func (f *FFmpeg) DecryptAAXC(inputPath, outputPath, key, iv string, progressCb func(ProgressInfo)) error {
	return f.DecryptAAXCWithMetadata(inputPath, outputPath, key, iv, Metadata{}, progressCb)
}

// DecryptAAXCWithMetadata decrypts AAXC and embeds metadata in one ffmpeg invocation.
func (f *FFmpeg) DecryptAAXCWithMetadata(inputPath, outputPath, key, iv string, meta Metadata, progressCb func(ProgressInfo)) error {
	decryptLog.Info().
		Str("input", inputPath).
		Str("output", outputPath).
		Msg("decrypting AAXC")

	err := f.runWithProgress(f.buildDecryptArgs(inputPath, outputPath, "", key, iv, meta), progressCb)
	if err != nil {
		decryptLog.Error().Err(err).Str("input", inputPath).Msg("AAXC decryption failed")
		return fmt.Errorf("AAXC decryption failed: %w", err)
	}

	decryptLog.Debug().Str("output", outputPath).Msg("AAXC decryption succeeded, validating")
	return f.validateDecryption(inputPath, outputPath, "")
}

func (f *FFmpeg) buildDecryptArgs(inputPath, outputPath, activationBytes, key, iv string, meta Metadata) []string {
	args := []string{}
	if activationBytes != "" {
		args = append(args, "-activation_bytes", activationBytes)
	}
	if key != "" && iv != "" {
		args = append(args, "-audible_key", key, "-audible_iv", iv)
	}

	args = append(args, "-i", inputPath)
	if meta.CoverPath != "" {
		args = append(args,
			"-i", meta.CoverPath,
			"-map", "0:a",
			"-map", "1:v",
			"-disposition:v:0", "attached_pic",
		)
	}

	args = append(args, "-c", "copy")
	args = append(args, buildMetadataArgs(meta)...)
	args = append(args, "-y", outputPath)
	return args
}

// validateDecryption checks that the output file has approximately the same duration.
func (f *FFmpeg) validateDecryption(inputPath, outputPath, activationBytes string) error {
	// Check file exists and has minimum size
	outInfo, err := os.Stat(outputPath)
	if err != nil {
		decryptLog.Error().Err(err).Str("output", outputPath).Msg("output file does not exist after decryption")
		return fmt.Errorf("output file not created: %w", err)
	}
	outSize := outInfo.Size()
	if outSize < 1024*100 { // At least 100KB for a valid audio file
		decryptLog.Error().Int64("size_bytes", outSize).Str("output", outputPath).Msg("output file too small, likely incomplete decryption")
		return fmt.Errorf("output file too small (%d bytes), decryption likely failed", outSize)
	}

	// Probe duration
	outDuration, err := f.Probe(outputPath)
	if err != nil {
		decryptLog.Error().Err(err).Str("output", outputPath).Msg("output validation probe failed")
		return fmt.Errorf("output validation failed: %w", err)
	}

	if outDuration < 60 {
		decryptLog.Warn().Float64("duration_sec", outDuration).Int64("size_bytes", outSize).Str("output", outputPath).Msg("output file suspiciously short")
		return fmt.Errorf("output file too short (%.1fs, %d bytes), decryption likely failed", outDuration, outSize)
	}

	decryptLog.Info().Float64("duration_sec", outDuration).Int64("size_bytes", outSize).Str("output", outputPath).Msg("decryption validated successfully")
	return nil
}

// Decrypt auto-detects the DRM type and decrypts accordingly.
func (f *FFmpeg) Decrypt(inputPath, outputPath, activationBytes, key, iv string) error {
	if key != "" && iv != "" {
		decryptLog.Debug().Str("input", inputPath).Msg("using AAXC decryption (key+iv)")
		return f.DecryptAAXC(inputPath, outputPath, key, iv, nil)
	}
	if activationBytes != "" {
		decryptLog.Debug().Str("input", inputPath).Msg("using AAX decryption (activation_bytes)")
		return f.DecryptAAX(inputPath, outputPath, activationBytes, nil)
	}
	decryptLog.Error().Str("input", inputPath).Msg("no decryption credentials available")
	return fmt.Errorf("no decryption credentials provided (need activation_bytes or key+iv)")
}

// ConvertToM4B converts a decrypted file to M4B (usually just a container copy).
func (f *FFmpeg) ConvertToM4B(inputPath, outputPath string) error {
	decryptLog.Info().Str("input", inputPath).Str("output", outputPath).Msg("converting to M4B")
	return f.run(
		"-i", inputPath,
		"-c", "copy",
		"-y",
		outputPath,
	)
}

// ConvertToMP3 converts a decrypted file to MP3.
func (f *FFmpeg) ConvertToMP3(inputPath, outputPath string, bitrate string) error {
	if bitrate == "" {
		bitrate = "128k"
	}
	decryptLog.Info().Str("input", inputPath).Str("output", outputPath).Str("bitrate", bitrate).Msg("converting to MP3")
	return f.run(
		"-i", inputPath,
		"-codec:a", "libmp3lame",
		"-b:a", bitrate,
		"-y",
		outputPath,
	)
}

// SplitChapters splits an audio file into separate chapter files.
func (f *FFmpeg) SplitChapters(inputPath, outputDir string, chapters []ChapterMark, format string) error {
	decryptLog.Info().Str("input", inputPath).Int("chapters", len(chapters)).Str("format", format).Msg("splitting chapters")
	ext := ".m4b"
	codec := []string{"-c", "copy"}
	if format == "mp3" {
		ext = ".mp3"
		codec = []string{"-codec:a", "libmp3lame", "-b:a", "128k"}
	}

	for i, ch := range chapters {
		outputPath := fmt.Sprintf("%s/%02d - %s%s", outputDir, i+1, sanitizeFilename(ch.Title), ext)
		args := []string{
			"-i", inputPath,
			"-ss", formatDuration(ch.StartMs),
		}
		if ch.EndMs > 0 {
			args = append(args, "-to", formatDuration(ch.EndMs))
		}
		args = append(args, codec...)
		args = append(args, "-y", outputPath)

		if err := f.run(args...); err != nil {
			return fmt.Errorf("split chapter %d (%s): %w", i+1, ch.Title, err)
		}
	}
	return nil
}

// ChapterMark represents a chapter boundary.
type ChapterMark struct {
	Title   string
	StartMs int
	EndMs   int
}

func formatDuration(ms int) string {
	totalSec := float64(ms) / 1000.0
	hours := int(totalSec) / 3600
	minutes := (int(totalSec) % 3600) / 60
	seconds := math.Mod(totalSec, 60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}

func sanitizeFilename(name string) string {
	replacer := []string{
		"<", "", ">", "", ":", "", "\"", "", "/", "", "\\", "", "|", "", "?", "", "*", "",
	}
	r := name
	for i := 0; i < len(replacer); i += 2 {
		r = replaceAll(r, replacer[i], replacer[i+1])
	}
	return r
}

func replaceAll(s, old, new string) string {
	for {
		i := indexOf(s, old)
		if i < 0 {
			return s
		}
		s = s[:i] + new + s[i+len(old):]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
