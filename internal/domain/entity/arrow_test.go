package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewArrowSelectUI(t *testing.T) {
	ui := NewArrowSelectUI()

	assert.Equal(t, ArrowSelectIdle, ui.State)
	assert.Equal(t, 10, ui.Config.MaxFrame)
	assert.Equal(t, 32, ui.Config.Radius)
	assert.Equal(t, 16, ui.Config.MinDistance)
	assert.Equal(t, 0, ui.Frame)
	assert.Equal(t, DirNone, ui.Highlighted)
}

func TestNewArrowSelectUIWithConfig(t *testing.T) {
	cfg := ArrowSelectConfig{
		Radius:      50,
		MinDistance: 20,
		MaxFrame:    15,
	}
	ui := NewArrowSelectUIWithConfig(cfg)

	assert.Equal(t, 50, ui.Config.Radius)
	assert.Equal(t, 20, ui.Config.MinDistance)
	assert.Equal(t, 15, ui.Config.MaxFrame)
}

func TestArrowSelectUI_IsActive(t *testing.T) {
	ui := NewArrowSelectUI()

	assert.False(t, ui.IsActive())

	ui.State = ArrowSelectAppearing
	assert.True(t, ui.IsActive())

	ui.State = ArrowSelectShown
	assert.True(t, ui.IsActive())

	ui.State = ArrowSelectDisappearing
	assert.True(t, ui.IsActive())
}

func TestArrowSelectUI_Update_Appearing(t *testing.T) {
	ui := NewArrowSelectUI()
	maxFrame := ui.Config.MaxFrame

	// Right click pressed - starts appearing
	ui.Update(true, false, 100, 100, 320, 240)
	assert.Equal(t, ArrowSelectAppearing, ui.State)
	assert.Equal(t, 0, ui.Frame)

	// Frame advances
	ui.Update(false, false, 100, 100, 320, 240)
	assert.Equal(t, 1, ui.Frame)

	// Advance to max frame
	for i := 0; i < maxFrame-1; i++ {
		ui.Update(false, false, 100, 100, 320, 240)
	}
	assert.Equal(t, ArrowSelectShown, ui.State)
	assert.Equal(t, maxFrame, ui.Frame)
}

func TestArrowSelectUI_Update_Disappearing(t *testing.T) {
	ui := NewArrowSelectUI()
	maxFrame := ui.Config.MaxFrame
	ui.State = ArrowSelectShown
	ui.Frame = maxFrame

	// Right click released - starts disappearing
	ui.Update(false, true, 100, 100, 320, 240)
	assert.Equal(t, ArrowSelectDisappearing, ui.State)
	assert.Equal(t, maxFrame, ui.Frame)

	// Frame decrements
	ui.Update(false, false, 100, 100, 320, 240)
	assert.Equal(t, maxFrame-1, ui.Frame)

	// Advance to 0
	for i := 0; i < maxFrame-1; i++ {
		ui.Update(false, false, 100, 100, 320, 240)
	}
	assert.Equal(t, ArrowSelectIdle, ui.State)
	assert.Equal(t, 0, ui.Frame)
}

func TestArrowSelectUI_Update_MidTransition(t *testing.T) {
	ui := NewArrowSelectUI()

	// Start appearing
	ui.Update(true, false, 100, 100, 320, 240)
	for i := 0; i < 5; i++ {
		ui.Update(false, false, 100, 100, 320, 240)
	}
	assert.Equal(t, ArrowSelectAppearing, ui.State)
	assert.Equal(t, 5, ui.Frame)

	// Release mid-animation - frame preserved
	ui.Update(false, true, 100, 100, 320, 240)
	assert.Equal(t, ArrowSelectDisappearing, ui.State)
	assert.Equal(t, 5, ui.Frame)

	// Press again - frame preserved
	ui.Update(true, false, 100, 100, 320, 240)
	assert.Equal(t, ArrowSelectAppearing, ui.State)
	assert.Equal(t, 5, ui.Frame)
}

func TestArrowSelectUI_UpdateHighlight(t *testing.T) {
	ui := NewArrowSelectUI()
	ui.CenterX = 100
	ui.CenterY = 100
	minDist := ui.Config.MinDistance

	tests := []struct {
		name     string
		mouseX   int
		mouseY   int
		expected Direction
	}{
		{"right", 100 + minDist + 10, 100, DirRight},
		{"up", 100, 100 - minDist - 10, DirUp},
		{"left", 100 - minDist - 10, 100, DirLeft},
		{"down", 100, 100 + minDist + 10, DirDown},
		{"too close", 100 + 5, 100 + 5, DirNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.UpdateHighlight(tt.mouseX, tt.mouseY)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expected, ui.Highlighted)
		})
	}
}

func TestArrowSelectUI_GetIconPosition(t *testing.T) {
	ui := NewArrowSelectUI()
	ui.CenterX = 100
	ui.CenterY = 100
	r := float64(ui.Config.Radius)

	tests := []struct {
		dir      Direction
		progress float64
		expectX  float64
		expectY  float64
	}{
		{DirRight, 1.0, 100 + r, 100},
		{DirUp, 1.0, 100, 100 - r},
		{DirLeft, 1.0, 100 - r, 100},
		{DirDown, 1.0, 100, 100 + r},
		{DirRight, 0.5, 100 + r*0.5, 100},
		{DirRight, 0.0, 100, 100},
	}

	for _, tt := range tests {
		x, y := ui.GetIconPosition(tt.dir, tt.progress)
		assert.Equal(t, tt.expectX, x)
		assert.Equal(t, tt.expectY, y)
	}
}

func TestArrowSelectUI_CenterClamp(t *testing.T) {
	ui := NewArrowSelectUI()
	r := ui.Config.Radius
	screenW, screenH := 320, 240

	// Near left edge
	ui.Update(true, false, 10, 100, screenW, screenH)
	assert.Equal(t, r, ui.CenterX)

	// Near right edge
	ui = NewArrowSelectUI()
	ui.Update(true, false, screenW-10, 100, screenW, screenH)
	assert.Equal(t, screenW-r, ui.CenterX)

	// Near top edge
	ui = NewArrowSelectUI()
	ui.Update(true, false, 100, 10, screenW, screenH)
	assert.Equal(t, r, ui.CenterY)

	// Near bottom edge
	ui = NewArrowSelectUI()
	ui.Update(true, false, 100, screenH-10, screenW, screenH)
	assert.Equal(t, screenH-r, ui.CenterY)
}

func TestArrowColors(t *testing.T) {
	// Verify all arrow types have colors defined
	assert.Contains(t, ArrowColors, ArrowGray)
	assert.Contains(t, ArrowColors, ArrowRed)
	assert.Contains(t, ArrowColors, ArrowBlue)
	assert.Contains(t, ArrowColors, ArrowPurple)

	// Verify gray is actually gray-ish
	gray := ArrowColors[ArrowGray]
	assert.Equal(t, gray.R, gray.G)
	assert.Equal(t, gray.G, gray.B)
}
