package entity

import "image/color"

// ArrowType represents the type of arrow
type ArrowType int

const (
	ArrowGray   ArrowType = iota // 기본 화살
	ArrowRed                     // 빨간색
	ArrowBlue                    // 파란색
	ArrowPurple                  // 보라색
)

// ArrowColors maps arrow types to their colors
var ArrowColors = map[ArrowType]color.RGBA{
	ArrowGray:   {128, 128, 128, 255},
	ArrowRed:    {255, 80, 80, 255},
	ArrowBlue:   {80, 80, 255, 255},
	ArrowPurple: {180, 80, 255, 255},
}

// ArrowSelectState represents the UI state
type ArrowSelectState int

const (
	ArrowSelectIdle ArrowSelectState = iota
	ArrowSelectAppearing
	ArrowSelectShown
	ArrowSelectDisappearing
)

// Direction represents a direction for arrow selection
type Direction int

const (
	DirNone  Direction = -1
	DirRight Direction = 0
	DirUp    Direction = 1
	DirLeft  Direction = 2
	DirDown  Direction = 3
)

// ArrowSelectConfig holds configuration for arrow selection UI
type ArrowSelectConfig struct {
	Radius      int // Icon distance from center (pixels)
	MinDistance int // Minimum distance for selection (pixels)
	MaxFrame    int // Animation duration (frames)
}

// DefaultArrowSelectConfig returns the default configuration
func DefaultArrowSelectConfig() ArrowSelectConfig {
	return ArrowSelectConfig{
		Radius:      32,
		MinDistance: 16,
		MaxFrame:    10,
	}
}

// ArrowSelectUI handles the arrow selection interface
type ArrowSelectUI struct {
	Config      ArrowSelectConfig
	State       ArrowSelectState
	Frame       int // Current animation frame (0~MaxFrame)
	CenterX     int // Right-click start position (clamped)
	CenterY     int
	Highlighted Direction // Currently highlighted direction (-1 = none)
}

// NewArrowSelectUI creates a new arrow selection UI with default config
func NewArrowSelectUI() *ArrowSelectUI {
	return NewArrowSelectUIWithConfig(DefaultArrowSelectConfig())
}

// NewArrowSelectUIWithConfig creates a new arrow selection UI with custom config
func NewArrowSelectUIWithConfig(cfg ArrowSelectConfig) *ArrowSelectUI {
	// Apply defaults for zero values
	if cfg.Radius == 0 {
		cfg.Radius = 32
	}
	if cfg.MinDistance == 0 {
		cfg.MinDistance = 16
	}
	if cfg.MaxFrame == 0 {
		cfg.MaxFrame = 10
	}

	return &ArrowSelectUI{
		Config:      cfg,
		State:       ArrowSelectIdle,
		Highlighted: DirNone,
	}
}

// IsActive returns true if the UI is visible
func (ui *ArrowSelectUI) IsActive() bool {
	return ui.State != ArrowSelectIdle
}

// GetProgress returns the animation progress (0.0 ~ 1.0)
func (ui *ArrowSelectUI) GetProgress() float64 {
	return float64(ui.Frame) / float64(ui.Config.MaxFrame)
}

// Update updates the UI state based on input
func (ui *ArrowSelectUI) Update(rightClickPressed, rightClickReleased bool, mouseX, mouseY, screenW, screenH int) {
	r := ui.Config.Radius
	maxFrame := ui.Config.MaxFrame

	switch ui.State {
	case ArrowSelectIdle:
		if rightClickPressed {
			ui.State = ArrowSelectAppearing
			ui.Frame = 0
			// Clamp center to keep icons on screen
			ui.CenterX = clamp(mouseX, r, screenW-r)
			ui.CenterY = clamp(mouseY, r, screenH-r)
		}

	case ArrowSelectAppearing:
		if rightClickReleased {
			// Transition to disappearing, keep frame
			ui.State = ArrowSelectDisappearing
		} else {
			ui.Frame++
			if ui.Frame >= maxFrame {
				ui.State = ArrowSelectShown
				ui.Frame = maxFrame
			}
		}

	case ArrowSelectShown:
		if rightClickReleased {
			ui.State = ArrowSelectDisappearing
			ui.Frame = maxFrame
		}

	case ArrowSelectDisappearing:
		if rightClickPressed {
			// Transition to appearing, keep frame
			ui.State = ArrowSelectAppearing
			// Update center position
			ui.CenterX = clamp(mouseX, r, screenW-r)
			ui.CenterY = clamp(mouseY, r, screenH-r)
		} else {
			ui.Frame--
			if ui.Frame <= 0 {
				ui.State = ArrowSelectIdle
				ui.Frame = 0
			}
		}
	}
}

// UpdateHighlight updates the highlighted direction based on mouse position
func (ui *ArrowSelectUI) UpdateHighlight(mouseX, mouseY int) Direction {
	deltaX := mouseX - ui.CenterX
	deltaY := mouseY - ui.CenterY

	// Minimum distance check (diamond shape)
	if abs(deltaX)+abs(deltaY) < ui.Config.MinDistance {
		ui.Highlighted = DirNone
		return DirNone
	}

	// Direction determination
	if abs(deltaX) >= abs(deltaY) {
		if deltaX > 0 {
			ui.Highlighted = DirRight
		} else {
			ui.Highlighted = DirLeft
		}
	} else {
		if deltaY < 0 {
			ui.Highlighted = DirUp
		} else {
			ui.Highlighted = DirDown
		}
	}

	return ui.Highlighted
}

// GetIconPosition returns the icon position for a direction
func (ui *ArrowSelectUI) GetIconPosition(dir Direction, easedProgress float64) (x, y float64) {
	radius := float64(ui.Config.Radius)
	var iconDX, iconDY float64

	switch dir {
	case DirRight:
		iconDX, iconDY = radius, 0
	case DirUp:
		iconDX, iconDY = 0, -radius
	case DirLeft:
		iconDX, iconDY = -radius, 0
	case DirDown:
		iconDX, iconDY = 0, radius
	}

	x = float64(ui.CenterX) + iconDX*easedProgress
	y = float64(ui.CenterY) + iconDY*easedProgress
	return x, y
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
