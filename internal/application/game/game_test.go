package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/younwookim/mg/internal/application/scene"
)

// mockScene is a test double for Scene interface
type mockScene struct {
	updateCalled int
	drawCalled   int
	onEnterCalled int
	onExitCalled  int
	nextScene    scene.Scene
	updateErr    error
}

func (m *mockScene) Update(dt float64) (scene.Scene, error) {
	m.updateCalled++
	return m.nextScene, m.updateErr
}

func (m *mockScene) Draw(screen *ebiten.Image) {
	m.drawCalled++
}

func (m *mockScene) OnEnter() {
	m.onEnterCalled++
}

func (m *mockScene) OnExit() {
	m.onExitCalled++
}

func TestNew(t *testing.T) {
	mockInitial := &mockScene{}
	g := New(mockInitial, 320, 240)

	assert.NotNil(t, g)
	assert.Equal(t, 1, mockInitial.onEnterCalled, "OnEnter should be called on initial scene")
}

func TestGame_Update_DelegatesToCurrentScene(t *testing.T) {
	mockInitial := &mockScene{}
	g := New(mockInitial, 320, 240)

	err := g.Update()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockInitial.updateCalled, "Update should delegate to current scene")
}

func TestGame_Draw_DelegatesToCurrentScene(t *testing.T) {
	mockInitial := &mockScene{}
	g := New(mockInitial, 320, 240)

	// Create a dummy image for testing
	img := ebiten.NewImage(320, 240)
	g.Draw(img)

	assert.Equal(t, 1, mockInitial.drawCalled, "Draw should delegate to current scene")
}

func TestGame_Layout(t *testing.T) {
	mockInitial := &mockScene{}
	g := New(mockInitial, 320, 240)

	w, h := g.Layout(640, 480)
	assert.Equal(t, 320, w)
	assert.Equal(t, 240, h)
}

func TestGame_SceneTransition(t *testing.T) {
	scene1 := &mockScene{}
	scene2 := &mockScene{}

	// scene1 will transition to scene2 on first update
	scene1.nextScene = scene2

	g := New(scene1, 320, 240)
	assert.Equal(t, 1, scene1.onEnterCalled, "Initial scene OnEnter called")

	// First update triggers transition
	err := g.Update()
	assert.NoError(t, err)

	assert.Equal(t, 1, scene1.updateCalled, "scene1 Update called")
	assert.Equal(t, 1, scene1.onExitCalled, "scene1 OnExit called on transition")
	assert.Equal(t, 1, scene2.onEnterCalled, "scene2 OnEnter called on transition")

	// Second update goes to scene2
	err = g.Update()
	assert.NoError(t, err)
	assert.Equal(t, 1, scene2.updateCalled, "scene2 Update called")
}

func TestGame_NoTransitionWhenNil(t *testing.T) {
	scene1 := &mockScene{nextScene: nil} // Returns nil, no transition

	g := New(scene1, 320, 240)

	// Multiple updates, no transition
	for i := 0; i < 5; i++ {
		err := g.Update()
		assert.NoError(t, err)
	}

	assert.Equal(t, 5, scene1.updateCalled, "All updates go to scene1")
	assert.Equal(t, 0, scene1.onExitCalled, "No OnExit when no transition")
}

func TestGame_UpdateError(t *testing.T) {
	scene1 := &mockScene{updateErr: assert.AnError}

	g := New(scene1, 320, 240)

	err := g.Update()
	assert.Error(t, err, "Error should propagate from scene")
}
