package playing

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/younwookim/mg/internal/application/replay"
	"github.com/younwookim/mg/internal/application/system"
)

// Recorder handles input recording for replay
type Recorder struct {
	data      replay.ReplayData
	filePath  string
	recording bool
	frame     int
}

// NewRecorder creates a new recorder
func NewRecorder(stage, filePath string) *Recorder {
	return &Recorder{
		data: replay.ReplayData{
			Version:   "1.0",
			Seed:      0, // Will be set later if needed
			Stage:     stage,
			StartTime: time.Now().Format(time.RFC3339),
			Frames:    make([]replay.FrameInput, 0, 3600),
		},
		filePath:  filePath,
		recording: true,
		frame:     0,
	}
}

// RecordFrame records a single frame's input
func (r *Recorder) RecordFrame(input system.InputState) {
	if !r.recording {
		return
	}

	frameInput := replay.FrameInput{
		F:   r.frame,
		L:   input.Left,
		R:   input.Right,
		U:   input.Up,
		D:   input.Down,
		J:   input.Jump,
		JP:  input.JumpPressed,
		JR:  input.JumpReleased,
		Dsh: input.Dash,
		MX:  input.MouseX,
		MY:  input.MouseY,
		MC:  input.MouseClick,
		RCP: input.RightClickPressed,
		RCR: input.RightClickReleased,
	}

	r.data.Frames = append(r.data.Frames, frameInput)
	r.frame++
}

// Save writes the replay data to a file
func (r *Recorder) Save() error {
	if len(r.data.Frames) == 0 {
		return nil // Nothing to save
	}

	file, err := os.Create(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r.data); err != nil {
		return fmt.Errorf("failed to encode replay: %w", err)
	}

	return nil
}

// Stop stops recording
func (r *Recorder) Stop() {
	r.recording = false
}

// IsRecording returns whether recording is active
func (r *Recorder) IsRecording() bool {
	return r.recording
}

// FrameCount returns the number of recorded frames
func (r *Recorder) FrameCount() int {
	return len(r.data.Frames)
}
