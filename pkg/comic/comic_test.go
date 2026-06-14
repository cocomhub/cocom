// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"encoding/json"
	"testing"
)

func TestNewComic(t *testing.T) {
	c := NewComic("1", "Test", []Image{{ID: "1", Path: "p1.jpg"}})
	if c.GetID() != "1" {
		t.Errorf("GetID() = %q, want %q", c.GetID(), "1")
	}
	if c.GetTitle() != "Test" {
		t.Errorf("GetTitle() = %q, want %q", c.GetTitle(), "Test")
	}
	if len(c.GetImages()) != 1 {
		t.Errorf("GetImages() len = %d, want 1", len(c.GetImages()))
	}
	if c.GetArchivePath() != "" {
		t.Errorf("GetArchivePath() = %q, want empty", c.GetArchivePath())
	}
}

func TestNewComicImplByObject(t *testing.T) {
	t.Run("ComicImpl value", func(t *testing.T) {
		original := ComicImpl{ID: "1", Title: "Test"}
		c, err := NewComicImplByObject(original)
		if err != nil {
			t.Fatalf("NewComicImplByObject failed: %v", err)
		}
		if c.ID != "1" {
			t.Errorf("ID = %q, want %q", c.ID, "1")
		}
	})
	t.Run("ComicImpl pointer", func(t *testing.T) {
		original := &ComicImpl{ID: "2", Title: "Test 2"}
		c, err := NewComicImplByObject(original)
		if err != nil {
			t.Fatalf("NewComicImplByObject failed: %v", err)
		}
		if c.ID != "2" {
			t.Errorf("ID = %q, want %q", c.ID, "2")
		}
	})
	t.Run("map[string]any", func(t *testing.T) {
		m := map[string]any{"id": "3", "title": "Test Map"}
		c, err := NewComicImplByObject(m)
		if err != nil {
			t.Fatalf("NewComicImplByObject failed: %v", err)
		}
		if c.ID != "3" {
			t.Errorf("ID = %q, want %q", c.ID, "3")
		}
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewComicImplByObject(nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})
	t.Run("invalid type", func(t *testing.T) {
		_, err := NewComicImplByObject(42)
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})
}

func TestComicImpl_MarshalJSON(t *testing.T) {
	c := &ComicImpl{
		ID:    "1",
		Title: "Test Comic",
		Tags:  []Tag{{ID: 1, Name: "tag1", Type: "tag"}},
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result["id"] != "1" {
		t.Errorf("id = %v, want %q", result["id"], "1")
	}
	if result["title"] != "Test Comic" {
		t.Errorf("title = %v, want %q", result["title"], "Test Comic")
	}
	// archivePath should not be in JSON output
	if _, ok := result["archivePath"]; ok {
		t.Error("archivePath should not be in JSON output")
	}
}

func TestComicImpl_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"id":"42","title":"Unmarshaled","images":[{"id":"img1","path":"p1.jpg","url":"http://example.com/p1.jpg"}]}`)
	var c ComicImpl
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if c.ID != "42" {
		t.Errorf("ID = %q, want %q", c.ID, "42")
	}
	if c.Title != "Unmarshaled" {
		t.Errorf("Title = %q, want %q", c.Title, "Unmarshaled")
	}
	if len(c.Images) != 1 {
		t.Fatalf("Images len = %d, want 1", len(c.Images))
	}
	if c.Images[0].ID != "img1" {
		t.Errorf("Image ID = %q, want %q", c.Images[0].ID, "img1")
	}
}

func TestComicImpl_JSONRoundTrip(t *testing.T) {
	original := &ComicImpl{
		ID:    "7",
		Title: "Round Trip",
		Images: []Image{
			{ID: "img1", Path: "p1.jpg", URL: "http://ex.com/p1.jpg"},
		},
		Tags: []Tag{
			{ID: 1, Name: "tag1", Type: "tag"},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var decoded ComicImpl
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
	}
	if len(decoded.Images) != len(original.Images) {
		t.Errorf("Images len = %d, want %d", len(decoded.Images), len(original.Images))
	}
}

func TestComicImpl_DefaultMethods(t *testing.T) {
	c := &ComicImpl{ID: "1", Title: "Test"}
	if c.GetTitleEnglish() != "" {
		t.Errorf("GetTitleEnglish() = %q, want empty", c.GetTitleEnglish())
	}
	if c.GetTitleJapanese() != "" {
		t.Errorf("GetTitleJapanese() = %q, want empty", c.GetTitleJapanese())
	}
	if c.GetTitlePretty() != "" {
		t.Errorf("GetTitlePretty() = %q, want empty", c.GetTitlePretty())
	}
	if c.IsStatus() {
		t.Error("IsStatus() should be false")
	}
	if c.IsDeleted() {
		t.Error("IsDeleted() should be false")
	}
	if c.GetRedirectCID() != 0 {
		t.Errorf("GetRedirectCID() = %d, want 0", c.GetRedirectCID())
	}
	if c.GetInvalidCount() != 0 {
		t.Errorf("GetInvalidCount() = %d, want 0", c.GetInvalidCount())
	}
	if c.GetFixedCount() != 0 {
		t.Errorf("GetFixedCount() = %d, want 0", c.GetFixedCount())
	}
}

func TestVerifyInfo_IsValid(t *testing.T) {
	v := &VerifyInfo{Valid: true}
	if !v.IsValid() {
		t.Error("IsValid() should be true")
	}
	v.Valid = false
	if v.IsValid() {
		t.Error("IsValid() should be false")
	}
}

func TestVerifyInfo_SetVerifyResult(t *testing.T) {
	v := &VerifyInfo{}
	r := &VerifyResult{
		Valid:                   true,
		InvalidCount:            3,
		InvalidSubsamplingCount: 1,
		FixedCount:              2,
	}
	v.SetVerifyResult(r)
	if !v.Valid {
		t.Error("Valid should be true")
	}
	if v.InvalidCount != 3 {
		t.Errorf("InvalidCount = %d, want 3", v.InvalidCount)
	}
	if v.InvalidSubsamplingCount != 1 {
		t.Errorf("InvalidSubsamplingCount = %d, want 1", v.InvalidSubsamplingCount)
	}
	if v.FixedCount != 2 {
		t.Errorf("FixedCount = %d, want 2", v.FixedCount)
	}
	// nil result should not panic
	v.SetVerifyResult(nil)
}

func TestVerifyInfo_Counters(t *testing.T) {
	v := &VerifyInfo{
		InvalidCount:            5,
		InvalidSubsamplingCount: 2,
		FixedCount:              3,
	}
	if v.GetInvalidCount() != 5 {
		t.Errorf("GetInvalidCount() = %d, want 5", v.GetInvalidCount())
	}
	if v.GetInvalidSubsamplingCount() != 2 {
		t.Errorf("GetInvalidSubsamplingCount() = %d, want 2", v.GetInvalidSubsamplingCount())
	}
	if v.GetFixedCount() != 3 {
		t.Errorf("GetFixedCount() = %d, want 3", v.GetFixedCount())
	}
}
