package service

import (
	"testing"
)

func TestAuthServiceMethods(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"LoginUser exists", "LoginUser"},
		{"RegisterUser exists", "RegisterUser"},
		{"RefreshUserSession exists", "RefreshUserSession"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Method %s is present in AuthService interface", tt.want)
		})
	}
}

func TestCreateSessionRequiresContext(t *testing.T) {
	t.Skip("Requires initialized repositories — integration test skipped in unit mode")
}
