package validator

import (
	"testing"
)

func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name standard", "Acme Corporation", false},
		{"valid name short", "Ab", false},
		{"valid unicode vietnamese", "Công Ty TNHH Một Thành Viên", false},
		{"invalid too short", "A", true},
		{"invalid empty", "", true},
		{"invalid only spaces", "    ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspaceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkspaceName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWorkspaceDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid domain standard", "acme-corp", false},
		{"valid domain with numbers", "dev123", false},
		{"valid domain hyphen middle", "my-awesome-team-1", false},
		{"invalid too short", "ac", true},
		{"invalid too long", "acme-corp-development-department-subdivision-of-the-entire-company", true},
		{"invalid start hyphen", "-acme", true},
		{"invalid end hyphen", "acme-", true},
		{"invalid uppercase", "Acme", true},
		{"invalid special char space", "acme corp", true},
		{"invalid special char dot", "acme.corp", true},
		{"invalid blacklist admin", "admin", true},
		{"invalid blacklist api", "api", true},
		{"invalid blacklist dev", "dev", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspaceDomain(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkspaceDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
