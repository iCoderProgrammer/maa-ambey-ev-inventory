package main

import (
	"strings"
	"testing"
)

func TestParsePolicy(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     string
		want    command
		wantErr string
	}{
		{"up", []string{"up"}, "production", command{name: "up"}, ""},
		{"version", []string{"version"}, "staging", command{name: "version"}, ""},
		{"down defaults to one step", []string{"down"}, "development", command{name: "down", steps: 1}, ""},
		{"down N", []string{"down", "2"}, "test", command{name: "down", steps: 2}, ""},
		{"down refused in production", []string{"down"}, "production", command{}, "refusing \"down\" with APP_ENV=production"},
		{"down refused in staging", []string{"down", "1"}, "staging", command{}, "--confirm"},
		{"down confirmed in production", []string{"down", "1", "--confirm"}, "production", command{name: "down", steps: 1}, ""},
		{"force refused in production", []string{"force", "1"}, "production", command{}, "refusing \"force\""},
		{"force confirmed", []string{"--confirm", "force", "1"}, "production", command{name: "force", version: 1}, ""},
		{"force in dev", []string{"force", "0"}, "development", command{name: "force"}, ""},
		{"force beyond latest", []string{"force", "999"}, "development", command{}, "between 0 and"},
		{"force negative", []string{"force", "-1"}, "development", command{}, "between 0 and"},
		{"down zero", []string{"down", "0"}, "development", command{}, "positive number"},
		{"down garbage", []string{"down", "all"}, "development", command{}, "positive number"},
		{"up with args", []string{"up", "5"}, "development", command{}, "takes no arguments"},
		{"no drop command", []string{"drop"}, "development", command{}, "unknown command"},
		{"missing command", nil, "development", command{}, "missing command"},
		{"bad app env", []string{"up"}, "prod", command{}, "APP_ENV must be one of"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.args, tt.env, 1)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got (%+v, %v), want %+v", got, err, tt.want)
			}
		})
	}
}

func TestRunRejectsBadConfigurationBeforeConnecting(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"missing url", map[string]string{}, "DATABASE_URL is required"},
		{"wrong scheme", map[string]string{"DATABASE_URL": "mysql://u:s3cret@localhost/db"}, "postgres:// or postgresql://"},
		{"no database", map[string]string{"DATABASE_URL": "postgres://u:s3cret@localhost:5432/"}, "must name a database"},
		{"insecure production", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://u:s3cret@db:5432/app?sslmode=disable"}, "sslmode=disable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			err := run([]string{"up"}, func(k string) string { return tt.env[k] }, &out)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
			if strings.Contains(err.Error(), "s3cret") {
				t.Errorf("error leaked the password: %v", err)
			}
		})
	}
}

func TestRedact(t *testing.T) {
	raw := "postgres://u:p%40ss%2Fw0rd@localhost/db"
	got := redact("open pgx5://u:p%40ss%2Fw0rd@localhost/db failed", raw)
	if strings.Contains(got, "w0rd") || !strings.Contains(got, "pgx5://u:xxxxx@localhost/db") {
		t.Errorf("password not redacted: %s", got)
	}
	// A short password must not mangle unrelated text.
	msg := "failed to open database: dial tcp 127.0.0.1:5999"
	if got := redact(msg, "postgres://u:p@localhost:5999/x"); got != msg {
		t.Errorf("redact changed unrelated text: %s", got)
	}
}
