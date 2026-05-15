package set

import "testing"

func Test_validatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{name: "valid port", port: "443", wantErr: false},
		{name: "valid max port", port: "65535", wantErr: false},
		{name: "valid min port", port: "1", wantErr: false},
		{name: "zero port", port: "0", wantErr: true},
		{name: "negative port", port: "-1", wantErr: true},
		{name: "port too large", port: "70000", wantErr: true},
		{name: "not a number", port: "abc", wantErr: true},
		{name: "empty string", port: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestSetClusterOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		o       *SetClusterOptions
		wantErr bool
	}{
		{
			name: "no flags changed",
			o: &SetClusterOptions{
				changedExternalIP:         false,
				changedExternalDomain:     false,
				changedExternalPort:       false,
				changedExternalDomainPort: false,
			},
			wantErr: true,
		},
		{
			name: "valid external IP",
			o: &SetClusterOptions{
				ExternalIP:        "1.2.3.4",
				changedExternalIP: true,
			},
			wantErr: false,
		},
		{
			name: "invalid external IP",
			o: &SetClusterOptions{
				ExternalIP:        "not-an-ip",
				changedExternalIP: true,
			},
			wantErr: true,
		},
		{
			name: "clear external IP with empty string",
			o: &SetClusterOptions{
				ExternalIP:        "",
				changedExternalIP: true,
			},
			wantErr: false,
		},
		{
			name: "valid external domain",
			o: &SetClusterOptions{
				ExternalDomain:        "api.example.com",
				changedExternalDomain: true,
			},
			wantErr: false,
		},
		{
			name: "invalid external domain",
			o: &SetClusterOptions{
				ExternalDomain:        "invalid!domain",
				changedExternalDomain: true,
			},
			wantErr: true,
		},
		{
			name: "valid external port",
			o: &SetClusterOptions{
				ExternalPort:        "8443",
				changedExternalPort: true,
			},
			wantErr: false,
		},
		{
			name: "invalid external port zero",
			o: &SetClusterOptions{
				ExternalPort:        "0",
				changedExternalPort: true,
			},
			wantErr: true,
		},
		{
			name: "invalid external port too large",
			o: &SetClusterOptions{
				ExternalPort:        "70000",
				changedExternalPort: true,
			},
			wantErr: true,
		},
		{
			name: "valid external domain port",
			o: &SetClusterOptions{
				ExternalDomainPort:        "8443",
				changedExternalDomainPort: true,
			},
			wantErr: false,
		},
		{
			name: "multiple flags changed",
			o: &SetClusterOptions{
				ExternalIP:          "1.2.3.4",
				ExternalPort:        "8443",
				changedExternalIP:   true,
				changedExternalPort: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.o.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
