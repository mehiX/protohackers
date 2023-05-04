package main

import "testing"

func TestValidateUsername(t *testing.T) {

	type scenario struct {
		username      string
		isValid       bool
		validUsername string
	}

	scenarios := []scenario{
		{username: "user1", isValid: true, validUsername: "user1"},
		{username: "  user1 ", isValid: true, validUsername: "user1"},
		{username: "useruseruseruser", isValid: true, validUsername: "useruseruseruser"},
		{username: "1234567890123456789", isValid: false},
		{username: "12345678", isValid: false},
		{username: "1234a5678", isValid: true, validUsername: "1234a5678"},
	}

	for _, s := range scenarios {
		t.Run(s.username, func(t *testing.T) {
			s := s
			t.Parallel()
			name, valid := validateUsername(s.username)
			if s.isValid != valid {
				t.Fatalf("should be: %v", s.isValid)
			}
			if valid && name != s.validUsername {
				t.Fatalf("wrong validated username. expected: %s, got: %s", s.validUsername, name)
			}
		})
	}
}
