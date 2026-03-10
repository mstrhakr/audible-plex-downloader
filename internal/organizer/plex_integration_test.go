package organizer

import (
	"context"
	"testing"

	"github.com/mstrhakr/audible-plex-downloader/internal/audnexus"
	"github.com/mstrhakr/audible-plex-downloader/internal/database"
)

// TestRealAudnexusNaming tests the naming functions against real Audnexus API data.
// This is an integration test that makes real API calls.
// Run from repo root with: go test -run TestRealAudnexusNaming ./internal/organizer/...
// Or all packages with: go test -run TestRealAudnexusNaming ./...
func TestRealAudnexusNaming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := audnexus.NewClient()
	ctx := context.Background()

	tests := []struct {
		name           string
		asin           string
		expectTitle    string
		expectSeries   bool
		expectSubtitle bool
		expectRegion   bool
	}{
		{
			name:         "Harry Potter series book",
			asin:         "B017V4IM1G",
			expectTitle:  "Harry Potter and the Sorcerer's Stone",
			expectSeries: true,
			expectRegion: true,
		},
		{
			name:         "Leviathan Wakes series book",
			asin:         "B073H9PF2D",
			expectTitle:  "Leviathan Wakes",
			expectSeries: true,
			expectRegion: true,
		},
		{
			name:         "The Martian standalone",
			asin:         "B082BHJMFF",
			expectTitle:  "The Martian",
			expectSeries: false,
			expectRegion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fetch real metadata from Audnexus
			book := &database.Book{ASIN: tt.asin}
			enriched, err := client.EnrichMetadata(ctx, book)
			if err != nil {
				t.Fatalf("failed to fetch metadata: %v", err)
			}

			// Verify we got expected data
			if enriched.Title() == "" {
				t.Error("expected title, got empty string")
			}
			if tt.expectTitle != "" && enriched.Title() != tt.expectTitle {
				t.Logf("note: expected title %q, got %q (API may have changed)", tt.expectTitle, enriched.Title())
			}

			if tt.expectSeries {
				if enriched.Series() == "" {
					t.Error("expected series name, got empty string")
				}
				if enriched.SeriesPosition() == "" {
					t.Error("expected series position, got empty string")
				}
			}

			if tt.expectRegion && enriched.Region() == "" {
				t.Error("expected region code, got empty string")
			}

			// Build the paths
			title := enriched.Title()
			subtitle := enriched.Subtitle()
			series := enriched.Series()
			seriesPosition := enriched.SeriesPosition()
			region := enriched.Region()

			filename := buildFilenameBase(title, subtitle, series, seriesPosition, tt.asin, region)
			bookDir := buildBookDirectoryName(title, tt.asin, region)

			// Log the actual output for inspection
			t.Logf("Title: %s", title)
			t.Logf("Subtitle: %s", subtitle)
			t.Logf("Series: %s", series)
			t.Logf("Series Position: %s", seriesPosition)
			t.Logf("Region: %s", region)
			t.Logf("Book Dir: %s", bookDir)
			t.Logf("Filename: %s", filename)

			// Verify basic structure
			if filename == "" {
				t.Error("filename should not be empty")
			}
			if bookDir == "" {
				t.Error("bookDir should not be empty")
			}

			// Verify ASIN is included
			if tt.asin != "" {
				if !contains(filename, tt.asin) {
					t.Errorf("filename should contain ASIN %s, got: %s", tt.asin, filename)
				}
				if !contains(bookDir, tt.asin) {
					t.Errorf("bookDir should contain ASIN %s, got: %s", tt.asin, bookDir)
				}
			}

			// Verify region is included if available
			if region != "" {
				expectedRegion := "[" + region + "]"
				if !contains(filename, expectedRegion) {
					t.Errorf("filename should contain region %s, got: %s", expectedRegion, filename)
				}
				if !contains(bookDir, expectedRegion) {
					t.Errorf("bookDir should contain region %s, got: %s", expectedRegion, bookDir)
				}
			}

			// Verify series is included if available
			if series != "" {
				if !contains(filename, series) {
					t.Errorf("filename should contain series %s, got: %s", series, filename)
				}
			}

			// Verify subtitle is included if available
			if subtitle != "" {
				if !contains(filename, subtitle) {
					t.Errorf("filename should contain subtitle %s, got: %s", subtitle, filename)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
