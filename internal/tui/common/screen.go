package common

// Screen identifies the currently active top-level view.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenTemplatePicker
)

// SwitchScreenMsg asks the root App to navigate between screens.
// Label is optional context passed to the target screen.
type SwitchScreenMsg struct {
	Target Screen
	Label  string
}
