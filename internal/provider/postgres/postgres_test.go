package postgres

import (
	"strings"
	"testing"
)

func TestValidateSnapshotName(t *testing.T) {
	const dbName = "myapp_development"

	// The longest derived identifier is the copy database; this is the
	// longest snapshot name that keeps it within the Postgres limit.
	maxName := strings.Repeat("x", maxIdentifierLength-len(snapshotCopyDatabaseName(dbName, "")))

	tests := []struct {
		name     string
		snapshot string
		wantErr  bool
	}{
		{name: "simple name", snapshot: "nightly", wantErr: false},
		{name: "double underscore", snapshot: "before__migration", wantErr: false},
		{name: "max length", snapshot: maxName, wantErr: false},
		{name: "empty", snapshot: "", wantErr: true},
		{name: "contains colon", snapshot: "before:migration", wantErr: true},
		{name: "contains legacy separator", snapshot: "before____migration", wantErr: true},
		{name: "too long", snapshot: maxName + "x", wantErr: true},
	}

	p := &Provider{config: &Config{DatabaseName: dbName}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateSnapshotName(tt.snapshot)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSnapshotName(%q) error = %v, wantErr %v", tt.snapshot, err, tt.wantErr)
			}
		})
	}
}

func TestParseSnapshotName(t *testing.T) {
	const dbName = "myapp_development"

	tests := []struct {
		name       string
		snapshotDB string
		want       string
		wantOK     bool
	}{
		{name: "new scheme", snapshotDB: "lunar:myapp_development:nightly", want: "nightly", wantOK: true},
		{name: "new scheme copy is skipped", snapshotDB: "lunar:myapp_development:nightly:copy", wantOK: false},
		{name: "legacy scheme", snapshotDB: "lunar_snapshot____myapp_development____nightly", want: "nightly", wantOK: true},
		{name: "legacy scheme copy is skipped", snapshotDB: "lunar_snapshot____myapp_development____nightly_copy", wantOK: false},
		{name: "other database is ignored", snapshotDB: "lunar:other_db:nightly", wantOK: false},
		{name: "unrelated database is ignored", snapshotDB: "myapp_development", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSnapshotName(tt.snapshotDB, dbName)
			if ok != tt.wantOK {
				t.Fatalf("parseSnapshotName(%q) ok = %v, want %v", tt.snapshotDB, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("parseSnapshotName(%q) = %q, want %q", tt.snapshotDB, got, tt.want)
			}
		})
	}
}
