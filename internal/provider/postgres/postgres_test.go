package postgres

import (
	"strings"
	"testing"
)

func TestValidateSnapshotName(t *testing.T) {
	const dbName = "myapp_development"

	// The longest derived identifier is the "_copy" database; this is the
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
		{name: "contains separator", snapshot: "before____migration", wantErr: true},
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
