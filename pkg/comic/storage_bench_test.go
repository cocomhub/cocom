// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMemoryStorage_Save(b *testing.B) {
	ctx := context.Background()
	sizes := []int{100, 1000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("Save-%d", n), func(b *testing.B) {
			for range b.N {
				b.StopTimer()
				ms := NewMemoryStorage()
				comics := make([]Comic, n)
				for i := range n {
					comics[i] = NewComic(fmt.Sprintf("%d", i), fmt.Sprintf("Comic %d", i), nil)
				}
				b.StartTimer()
				for _, c := range comics {
					if err := ms.Save(ctx, c); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkMemoryStorage_Get(b *testing.B) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	n := 500
	for i := range n {
		c := NewComic(fmt.Sprintf("%d", i), fmt.Sprintf("Comic %d", i), nil)
		if err := ms.Save(ctx, c); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		id := fmt.Sprintf("%d", n/2)
		_, err := ms.Get(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorage_Find(b *testing.B) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	n := 500
	for i := range n {
		c := NewComic(fmt.Sprintf("%d", i), fmt.Sprintf("Comic %d", i), nil)
		if err := ms.Save(ctx, c); err != nil {
			b.Fatal(err)
		}
	}

	filter := &ComicFilter{}
	filter.SetLimit(50)

	b.ResetTimer()
	for range b.N {
		_, err := ms.Find(ctx, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorage_FindTotal(b *testing.B) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	n := 500
	for i := range n {
		c := NewComic(fmt.Sprintf("%d", i), fmt.Sprintf("Comic %d", i), nil)
		if err := ms.Save(ctx, c); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		_, err := ms.FindTotal(ctx, &ComicFilter{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorage_SearchTags(b *testing.B) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	// Seed comics with tags
	for i := range 200 {
		c := NewComic(fmt.Sprintf("%d", i), fmt.Sprintf("Comic %d", i), nil)
		tags := make([]Tag, 10)
		for j := range 10 {
			tags[j] = Tag{
				ID:   j + 1,
				Name: fmt.Sprintf("tag-%d-%d", i, j),
				Type: "tag",
			}
		}
		// Use ComicImpl to set tags
		if impl, ok := c.(*ComicImpl); ok {
			impl.Tags = tags
		}
		if err := ms.Save(ctx, c); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.Run("SearchTags-exact", func(b *testing.B) {
		for range b.N {
			_, _, err := ms.SearchTags(ctx, "tag", "tag-100-5", 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("SearchTags-pattern", func(b *testing.B) {
		for range b.N {
			_, _, err := ms.SearchTags(ctx, "tag", "tag-1", 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
