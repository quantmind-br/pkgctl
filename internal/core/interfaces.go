package core

// InstallOptions contains options for package installation
type InstallOptions struct {
	SkipDesktop    bool   // Skip desktop integration
	CustomName     string // Custom application name
	SkipWaylandEnv bool   // Skip Wayland environment variable injection
}
