// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion_PrintVersion(t *testing.T) {
	var buf bytes.Buffer
	n, err := PrintVersion(&buf, "%{Version}|%{Branch}|%{CommitID}|%{DirtyID}")
	if err != nil {
		t.Fatalf("PrintVersion failed: %v", err)
	}
	if n <= 0 {
		t.Errorf("PrintVersion returned n=%d, want > 0", n)
	}
	parts := strings.SplitN(strings.TrimSpace(buf.String()), "|", 4)
	if len(parts) != 4 {
		t.Fatalf("unexpected output: %q", buf.String())
	}
	if parts[0] != Version {
		t.Errorf("Version field = %q, want %q", parts[0], Version)
	}
	if parts[3] != DirtyID {
		t.Errorf("DirtyID field = %q, want %q", parts[3], DirtyID)
	}
}

func TestVersion_PrintVersion_DefaultFormat(t *testing.T) {
	var buf bytes.Buffer
	if _, err := PrintVersion(&buf, ""); err != nil {
		t.Fatalf("PrintVersion default format failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Version:") {
		t.Errorf("default format should contain 'Version:', got %q", out)
	}
}

func TestVersion_PrintVersionJSON(t *testing.T) {
	var buf bytes.Buffer
	if _, err := PrintVersionJSON(&buf); err != nil {
		t.Fatalf("PrintVersionJSON failed: %v", err)
	}
	if !strings.Contains(buf.String(), `"Version"`) {
		t.Errorf("json output should contain Version key, got %q", buf.String())
	}
}
