package replay

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
