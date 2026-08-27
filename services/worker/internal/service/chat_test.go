package service

import (
	"testing"
)

func TestChatServiceInterface(t *testing.T) {
	var svc ChatService
	_ = svc
	t.Log("ChatService interface available")
}

func TestChatServiceRequiresSessionID(t *testing.T) {
	t.Skip("Requires initialized repositories — integration test skipped in unit mode")
}
