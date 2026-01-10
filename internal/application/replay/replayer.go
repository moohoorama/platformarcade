package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ReplayInput represents input state during replay
type ReplayInput struct {
	Left               bool
	Right              bool
	Up                 bool
	Down               bool
	Jump               bool
	JumpPressed        bool
	JumpReleased       bool
	Dash               bool
	MouseX             int
	MouseY             int
	MouseClick         bool
	RightClickPressed  bool
	RightClickReleased bool
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
	defer func() { _ = file.Close() }()

	var data ReplayData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode replay: %w", err)
	}

	return &data, nil
}

// GetInput returns the input for the current frame and advances
func (r *Replayer) GetInput() (ReplayInput, bool) {
	if r.frame >= len(r.data.Frames) {
		return ReplayInput{}, false
	}

	fi := r.data.Frames[r.frame]
	r.frame++

	return ReplayInput{
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
