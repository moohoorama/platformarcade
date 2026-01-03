package config

// PhysicsConfig is the root config for physics.json
type PhysicsConfig struct {
	Display     DisplayConfig     `json:"display"`
	Physics     PhysicsSettings   `json:"physics"`
	Movement    MovementConfig    `json:"movement"`
	Jump        JumpConfig        `json:"jump"`
	Dash        DashConfig        `json:"dash"`
	Collision   CollisionConfig   `json:"collision"`
	Combat      CombatConfig      `json:"combat"`
	Feedback    FeedbackConfig    `json:"feedback"`
	ArrowSelect        ArrowSelectConfig        `json:"arrowSelect"`
	Projectile         ProjectileBehaviorConfig `json:"projectile"`
}

// ArrowSelectConfig configures the arrow selection UI
type ArrowSelectConfig struct {
	Radius      int `json:"radius"`      // Icon distance from center (pixels)
	MinDistance int `json:"minDistance"` // Minimum distance for selection (pixels)
	MaxFrame    int `json:"maxFrame"`    // Animation duration (frames)
}

type DisplayConfig struct {
	ScreenWidth  int `json:"screenWidth"`
	ScreenHeight int `json:"screenHeight"`
	Scale        int `json:"scale"`
	Framerate    int `json:"framerate"`
}

type PhysicsSettings struct {
	Substeps           int     `json:"substeps"`
	Gravity            float64 `json:"gravity"`
	MaxFallSpeed       float64 `json:"maxFallSpeed"`
	UseIntegerPosition bool    `json:"useIntegerPosition"`
}

type MovementConfig struct {
	Acceleration    float64 `json:"acceleration"`
	Deceleration    float64 `json:"deceleration"`
	MaxSpeed        float64 `json:"maxSpeed"`
	AirControl      float64 `json:"airControl"`
	TurnaroundBoost float64 `json:"turnaroundBoost"`
}

type JumpConfig struct {
	Force                  float64           `json:"force"`
	VariableJumpMultiplier float64           `json:"variableJumpMultiplier"`
	CoyoteTime             float64           `json:"coyoteTime"`
	JumpBuffer             float64           `json:"jumpBuffer"`
	ApexModifier           ApexModifierConfig `json:"apexModifier"`
	FallMultiplier         float64           `json:"fallMultiplier"`
}

type ApexModifierConfig struct {
	Enabled           bool    `json:"enabled"`
	Threshold         float64 `json:"threshold"`
	GravityMultiplier float64 `json:"gravityMultiplier"`
	SpeedBoost        float64 `json:"speedBoost"`
}

type DashConfig struct {
	Speed           float64 `json:"speed"`
	Duration        float64 `json:"duration"`
	Cooldown        float64 `json:"cooldown"`
	IframesDuration float64 `json:"iframesDuration"`
}

type CollisionConfig struct {
	CornerCorrection MarginConfig `json:"cornerCorrection"`
	LedgeAssist      MarginConfig `json:"ledgeAssist"`
}

type MarginConfig struct {
	Enabled bool `json:"enabled"`
	Margin  int  `json:"margin"`
}

type CombatConfig struct {
	Iframes   float64        `json:"iframes"`
	Knockback KnockbackConfig `json:"knockback"`
}

type KnockbackConfig struct {
	Force        float64 `json:"force"`
	UpForce      float64 `json:"upForce"`
	StunDuration float64 `json:"stunDuration"`
}

type FeedbackConfig struct {
	Hitstop       HitstopConfig       `json:"hitstop"`
	ScreenShake   ScreenShakeConfig   `json:"screenShake"`
	SquashStretch SquashStretchConfig `json:"squashStretch"`
}

type HitstopConfig struct {
	Enabled bool `json:"enabled"`
	Frames  int  `json:"frames"`
}

type ScreenShakeConfig struct {
	Enabled   bool    `json:"enabled"`
	Intensity float64 `json:"intensity"`
	Decay     float64 `json:"decay"`
}

type SquashStretchConfig struct {
	Enabled    bool      `json:"enabled"`
	LandSquash ScaleXY   `json:"landSquash"`
	JumpStretch ScaleXY  `json:"jumpStretch"`
	Duration   float64   `json:"duration"`
}

type ScaleXY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ProjectileBehaviorConfig configures projectile physics behavior
type ProjectileBehaviorConfig struct {
	// VelocityInfluence controls how much player velocity affects arrow velocity
	// 0.0 = no influence (arrow fires at fixed speed)
	// 1.0 = full influence (player velocity is fully added to arrow)
	// 0.5 = partial influence (50% of player velocity is added)
	VelocityInfluence float64 `json:"velocityInfluence"`
}
