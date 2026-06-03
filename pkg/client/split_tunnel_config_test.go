package client

import (
	"testing"
)

func TestSplitTunnelConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SplitTunnelConfig
		wantErr bool
	}{
		{
			name: "valid exclude mode",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"192.168.0.0/16"},
			},
			wantErr: false,
		},
		{
			name: "valid include mode",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeInclude,
				IncludeCIDRs: []string{"8.8.8.8/32"},
			},
			wantErr: false,
		},
		{
			name: "disabled config is valid",
			config: SplitTunnelConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "exclude mode without CIDRs",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{},
			},
			wantErr: true,
		},
		{
			name: "include mode without CIDRs",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeInclude,
				IncludeCIDRs: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid CIDR format",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"invalid-cidr"},
			},
			wantErr: true,
		},
		{
			name: "valid IPv6 CIDR",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"fd00::/8"},
			},
			wantErr: false,
		},
		{
			name: "multiple valid CIDRs",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    SplitTunnelModeExclude,
				ExcludeCIDRs: []string{
					"192.168.0.0/16",
					"10.0.0.0/8",
					"172.16.0.0/12",
				},
			},
			wantErr: false,
		},
		{
			name: "one invalid CIDR among valid ones",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    SplitTunnelModeExclude,
				ExcludeCIDRs: []string{
					"192.168.0.0/16",
					"invalid",
					"10.0.0.0/8",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSplitTunnelConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   SplitTunnelConfig
		wantMode SplitTunnelMode
	}{
		{
			name:     "empty mode sets to disabled",
			config:   SplitTunnelConfig{},
			wantMode: SplitTunnelModeDisabled,
		},
		{
			name: "existing mode is preserved",
			config: SplitTunnelConfig{
				Mode: SplitTunnelModeExclude,
			},
			wantMode: SplitTunnelModeExclude,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			if tt.config.Mode != tt.wantMode {
				t.Errorf("SetDefaults() mode = %v, want %v", tt.config.Mode, tt.wantMode)
			}
		})
	}
}
