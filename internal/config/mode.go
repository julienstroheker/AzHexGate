package config

// Mode represents the operational mode of the application
type Mode string

const (
	// ModeLocal runs with in-memory relay (no Azure required)
	ModeLocal Mode = "local"

	// ModeRemote runs with Azure Relay
	ModeRemote Mode = "remote"
)

// IsValid checks if the mode is valid
func (m Mode) IsValid() bool {
	return m == ModeLocal || m == ModeRemote
}

// String returns the string representation
func (m Mode) String() string {
	return string(m)
}
