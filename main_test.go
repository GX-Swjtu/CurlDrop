package main

import (
	"os"
	"testing"
)

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		setEnv     bool
		defaultVal int
		want       int
	}{
		{"valid value", "TEST_INT", "9090", true, 8080, 9090},
		{"invalid value falls back to default", "TEST_INT", "notanumber", true, 8080, 8080},
		{"empty value falls back to default", "TEST_INT", "", true, 8080, 8080},
		{"unset falls back to default", "TEST_INT_UNSET", "", false, 8080, 8080},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envKey, tt.envVal)
			} else {
				os.Unsetenv(tt.envKey)
			}
			got := getEnvInt(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvInt(%q, %d) = %d, want %d", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetEnvInt64(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		setEnv     bool
		defaultVal int64
		want       int64
	}{
		{"valid value", "TEST_INT64", "86400", true, 0, 86400},
		{"invalid value falls back to default", "TEST_INT64", "abc", true, 7, 7},
		{"unset falls back to default", "TEST_INT64_UNSET", "", false, 7, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envKey, tt.envVal)
			} else {
				os.Unsetenv(tt.envKey)
			}
			got := getEnvInt64(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvInt64(%q, %d) = %d, want %d", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetEnvStr(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		setEnv     bool
		defaultVal string
		want       string
	}{
		{"returns env value", "TEST_STR", "custom", true, "default", "custom"},
		{"empty value falls back to default", "TEST_STR", "", true, "default", "default"},
		{"unset falls back to default", "TEST_STR_UNSET", "", false, "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envKey, tt.envVal)
			} else {
				os.Unsetenv(tt.envKey)
			}
			got := getEnvStr(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvStr(%q, %q) = %q, want %q", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		setEnv     bool
		defaultVal bool
		want       bool
	}{
		{"true", "TEST_BOOL", "true", true, false, true},
		{"TRUE", "TEST_BOOL", "TRUE", true, false, true},
		{"1", "TEST_BOOL", "1", true, false, true},
		{"yes", "TEST_BOOL", "yes", true, false, true},
		{"Yes", "TEST_BOOL", "Yes", true, false, true},
		{"false", "TEST_BOOL", "false", true, true, false},
		{"0", "TEST_BOOL", "0", true, true, false},
		{"no", "TEST_BOOL", "no", true, true, false},
		{"random string", "TEST_BOOL", "random", true, true, false},
		{"unset default true", "TEST_BOOL_UNSET", "", false, true, true},
		{"unset default false", "TEST_BOOL_UNSET", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envKey, tt.envVal)
			} else {
				os.Unsetenv(tt.envKey)
			}
			got := getEnvBool(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvBool(%q, %v) = %v, want %v", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}
