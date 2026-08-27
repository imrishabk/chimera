package service

import (
	"testing"
)

func TestCreateSessionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Integration test enabled — requires live DB + Python gRPC")
}
