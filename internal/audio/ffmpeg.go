package audio

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
)

// FFmpeg wraps the ffmpeg and ffprobe binaries for audio processing.
type FFmpeg struct {
	binPath   string
	probePath string
}

// ProgressInfo holds parsed ffmpeg progress state from `-progress pipe:1`.
type ProgressInfo struct {
	Frame      int
	FPS        float64
	Bitrate    string
	TotalSize  int64
	OutTimeMs  int64
	OutTime    string
	DupFrames  int
	DropFrames int
	Speed      string
	Progress   string // e.g. "continue", "end"
}

var ffmpegLog = logging.Component("ffmpeg")

// NewFFmpeg locates or downloads ffmpeg/ffprobe and returns a ready wrapper.
// It checks the system PATH first, then {configDir}/bin/, downloading static
// builds from GitHub as a last resort.
func NewFFmpeg(configDir string) (*FFmpeg, error) {
	ffmpegPath, ffprobePath, err := ensureFFmpeg(configDir)
	if err != nil {
		return nil, err
	}
	ffmpegLog.Info().Str("ffmpeg", ffmpegPath).Str("ffprobe", ffprobePath).Msg("ffmpeg ready")
	return &FFmpeg{binPath: ffmpegPath, probePath: ffprobePath}, nil
}

// run executes an ffmpeg command and returns combined output on error.
func (f *FFmpeg) run(args ...string) error {
	cmd := exec.Command(f.binPath, args...)
	ffmpegLog.Debug().Strs("args", args).Msg("running ffmpeg")

	output, err := cmd.CombinedOutput()
	if err != nil {
		ffmpegLog.Error().Err(err).Str("output", strings.TrimSpace(string(output))).Msg("ffmpeg command failed")
		return fmt.Errorf("ffmpeg failed: %w\noutput: %s", err, strings.TrimSpace(string(output)))
	}
	ffmpegLog.Trace().Msg("ffmpeg command succeeded")
	return nil
}

// runWithProgress executes ffmpeg with `-progress pipe:1` and streams parsed progress.
func (f *FFmpeg) runWithProgress(args []string, cb func(ProgressInfo)) error {
	cmdArgs := append([]string{}, args...)
	cmdArgs = append(cmdArgs, "-progress", "pipe:1", "-nostats")

	cmd := exec.Command(f.binPath, cmdArgs...)
	ffmpegLog.Debug().Strs("args", cmdArgs).Msg("running ffmpeg with progress")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stderrBuf, stderr)
		close(stderrDone)
	}()

	var info ProgressInfo
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		switch key {
		case "frame":
			info.Frame = parseInt(val)
		case "fps":
			info.FPS = parseFloat(val)
		case "bitrate":
			info.Bitrate = val
		case "total_size":
			info.TotalSize = parseInt64(val)
		case "out_time_ms":
			info.OutTimeMs = parseInt64(val)
		case "out_time":
			info.OutTime = val
		case "dup_frames":
			info.DupFrames = parseInt(val)
		case "drop_frames":
			info.DropFrames = parseInt(val)
		case "speed":
			info.Speed = val
		case "progress":
			info.Progress = val
			if cb != nil {
				cb(info)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ffmpeg progress scan: %w", err)
	}

	waitErr := cmd.Wait()
	<-stderrDone
	if waitErr != nil {
		stderrText := strings.TrimSpace(stderrBuf.String())
		ffmpegLog.Error().Err(waitErr).Str("stderr", stderrText).Msg("ffmpeg command failed")
		if stderrText != "" {
			return fmt.Errorf("ffmpeg failed: %w\noutput: %s", waitErr, stderrText)
		}
		return fmt.Errorf("ffmpeg failed: %w", waitErr)
	}

	ffmpegLog.Trace().Msg("ffmpeg command with progress succeeded")
	return nil
}

// Probe returns the duration of an audio file in seconds.
func (f *FFmpeg) Probe(inputPath string) (float64, error) {
	ffmpegLog.Debug().Str("input", inputPath).Msg("probing audio file")

	cmd := exec.Command(f.probePath,
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		ffmpegLog.Error().Err(err).Str("input", inputPath).Msg("ffprobe failed")
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var duration float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &duration)
	ffmpegLog.Debug().Float64("duration_sec", duration).Str("input", inputPath).Msg("probe complete")
	return duration, err
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
