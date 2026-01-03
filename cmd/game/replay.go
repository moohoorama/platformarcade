package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/younwookim/mg/internal/application/system"
)

// FrameInput records input state for a single frame
type FrameInput struct {
	F   int  `json:"f"`             // Frame number
	L   bool `json:"l,omitempty"`   // Left
	R   bool `json:"r,omitempty"`   // Right
	U   bool `json:"u,omitempty"`   // Up
	D   bool `json:"d,omitempty"`   // Down
	J   bool `json:"j,omitempty"`   // Jump
	JP  bool `json:"jp,omitempty"`  // JumpPressed
	JR  bool `json:"jr,omitempty"`  // JumpReleased
	Dsh bool `json:"dsh,omitempty"` // Dash
	MX  int  `json:"mx"`            // MouseX
	MY  int  `json:"my"`            // MouseY
	MC  bool `json:"mc,omitempty"`  // MouseClick
	RCP bool `json:"rcp,omitempty"` // RightClickPressed
	RCR bool `json:"rcr,omitempty"` // RightClickReleased
}

// ReplayData contains all data needed to replay a game session
type ReplayData struct {
	Version   string       `json:"version"`
	Seed      int64        `json:"seed"`
	Stage     string       `json:"stage"`
	StartTime string       `json:"startTime"`
	Frames    []FrameInput `json:"frames"`
}

// Recorder handles input recording
type Recorder struct {
	data      ReplayData
	recording bool
	frame     int
}

// NewRecorder creates a new recorder
func NewRecorder(seed int64, stage string) *Recorder {
	return &Recorder{
		data: ReplayData{
			Version:   "1.0",
			Seed:      seed,
			Stage:     stage,
			StartTime: time.Now().Format(time.RFC3339),
			Frames:    make([]FrameInput, 0, 3600), // Pre-allocate for ~1 minute at 60fps
		},
		recording: true,
		frame:     0,
	}
}

// RecordFrame records a single frame's input
func (r *Recorder) RecordFrame(input system.InputState) {
	if !r.recording {
		return
	}

	frameInput := FrameInput{
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
	defer file.Close()

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

// GenerateFilename creates a filename based on current time
func GenerateFilename() string {
	return fmt.Sprintf("replay_%s.json", time.Now().Format("20060102_150405"))
}

// Replayer handles input playback from recorded data
type Replayer struct {
	data  ReplayData
	frame int
}

// NewReplayer creates a new replayer from replay data
func NewReplayer(data ReplayData) *Replayer {
	return &Replayer{
		data:  data,
		frame: 0,
	}
}

// LoadReplay loads replay data from a file
func LoadReplay(filename string) (*ReplayData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data ReplayData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode replay: %w", err)
	}

	return &data, nil
}

// GetInput returns the input for the current frame and advances
func (r *Replayer) GetInput() (system.InputState, bool) {
	if r.frame >= len(r.data.Frames) {
		return system.InputState{}, false
	}

	fi := r.data.Frames[r.frame]
	r.frame++

	return system.InputState{
		Left:               fi.L,
		Right:              fi.R,
		Up:                 fi.U,
		Down:               fi.D,
		Jump:               fi.J,
		JumpPressed:        fi.JP,
		JumpReleased:       fi.JR,
		Dash:               fi.Dsh,
		MouseX:             fi.MX,
		MouseY:             fi.MY,
		MouseClick:         fi.MC,
		RightClickPressed:  fi.RCP,
		RightClickReleased: fi.RCR,
	}, true
}

// CurrentFrame returns the current frame number
func (r *Replayer) CurrentFrame() int {
	return r.frame
}

// TotalFrames returns the total number of frames
func (r *Replayer) TotalFrames() int {
	return len(r.data.Frames)
}

// Seed returns the seed used for the replay
func (r *Replayer) Seed() int64 {
	return r.data.Seed
}

// Reset resets the replayer to the beginning
func (r *Replayer) Reset() {
	r.frame = 0
}

// CreateTestReplayData creates replay data for testing (idle player)
func CreateTestReplayData(frames int, mouseX, mouseY int) ReplayData {
	data := ReplayData{
		Version:   "1.0",
		Seed:      12345,
		Stage:     "test",
		StartTime: time.Now().Format(time.RFC3339),
		Frames:    make([]FrameInput, frames),
	}

	for i := 0; i < frames; i++ {
		data.Frames[i] = FrameInput{
			F:  i,
			MX: mouseX,
			MY: mouseY,
		}
	}

	return data
}
