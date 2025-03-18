package main

func ContainsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && (str[:len(substr)] == substr || ContainsIgnoreCase(str[1:], substr))
}
