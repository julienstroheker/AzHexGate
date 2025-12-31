package config

import "testing"

func TestMode_IsValid(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want bool
	}{
		{
			name: "local mode is valid",
			mode: ModeLocal,
			want: true,
		},
		{
			name: "remote mode is valid",
			mode: ModeRemote,
			want: true,
		},
		{
			name: "invalid mode",
			mode: Mode("invalid"),
			want: false,
		},
		{
			name: "empty mode",
			mode: Mode(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.IsValid(); got != tt.want {
				t.Errorf("Mode.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want string
	}{
		{
			name: "local mode string",
			mode: ModeLocal,
			want: "local",
		},
		{
			name: "remote mode string",
			mode: ModeRemote,
			want: "remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("Mode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
