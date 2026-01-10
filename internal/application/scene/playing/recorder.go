package playing

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/younwookim/mg/internal/application/replay"
)

// RecordableInput is the input interface for recording
type RecordableInput struct {
	Left, Right, Up, Down bool
	Jump                  bool
	JumpPressed           bool
	JumpReleased          bool
	Dash                  bool
	MouseX, MouseY        int
	MouseClick            bool
	RightClickPressed     bool
	RightClickReleased    bool
}

// Recorder handles input recording for replay
type Recorder struct {
	data      replay.ReplayData
	recording bool
	frame     int
}

// NewRecorder creates a new recorder with seed for deterministic replay
func NewRecorder(seed int64, stage string) *Recorder {
	return &Recorder{
		data: replay.ReplayData{
			Version:   "1.0",
			Seed:      seed,
			Stage:     stage,
			StartTime: time.Now().Format(time.RFC3339),
			Frames:    make([]replay.FrameInput, 0, 3600), // Pre-allocate for ~1 minute at 60fps
		},
		recording: true,
		frame:     0,
	}
}

// RecordFrame records a single frame's input
func (r *Recorder) RecordFrame(input RecordableInput) {
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
func (r *Recorder) Save(filename string) error {
	if len(r.data.Frames) == 0 {
		return fmt.Errorf("no frames to save")
	}

	file, err := os.Create(filename)
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

// GetData returns the replay data (for testing)
func (r *Recorder) GetData() replay.ReplayData {
	return r.data
}

// GenerateFilename creates a filename based on current time
func GenerateFilename() string {
	return fmt.Sprintf("replay_%s.json", time.Now().Format("20060102_150405"))
}
