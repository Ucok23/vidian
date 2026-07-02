package git

import (
	"testing"
	"time"
)

func TestGetActivityDates(t *testing.T) {
	dates, err := GetActivityDates(testDir)
	if err != nil {
		t.Fatalf("GetActivityDates returned error: %v", err)
	}

	// Vidian has commits, so we expect at least one date back.
	if len(dates) == 0 {
		t.Fatal("expected at least one activity date, got none")
	}

	// Every entry must be a valid YYYY-MM-DD date and within the last year.
	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, -1) // one day of slack
	for _, d := range dates {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil {
			t.Errorf("date %q is not in YYYY-MM-DD format: %v", d, err)
			continue
		}
		if parsed.Before(oneYearAgo) {
			t.Errorf("date %q is older than one year", d)
		}
	}
}

func TestGetHotFiles(t *testing.T) {
	files, err := GetHotFiles(testDir, 10)
	if err != nil {
		t.Fatalf("GetHotFiles returned error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one hot file, got none")
	}

	if len(files) > 10 {
		t.Errorf("expected at most 10 files, got %d", len(files))
	}

	// Results must be sorted descending by commit count.
	for i := 1; i < len(files); i++ {
		if files[i].Commits > files[i-1].Commits {
			t.Errorf("files not sorted: files[%d].Commits=%d > files[%d].Commits=%d",
				i, files[i].Commits, i-1, files[i-1].Commits)
		}
	}

	// Every entry must have a non-empty path and a positive commit count.
	for _, f := range files {
		if f.Path == "" {
			t.Error("found hot file with empty path")
		}
		if f.Commits <= 0 {
			t.Errorf("file %q has non-positive commit count %d", f.Path, f.Commits)
		}
	}
}

func TestGetHotFilesLimit(t *testing.T) {
	// Requesting 3 should never return more than 3.
	files, err := GetHotFiles(testDir, 3)
	if err != nil {
		t.Fatalf("GetHotFiles(3) returned error: %v", err)
	}
	if len(files) > 3 {
		t.Errorf("expected at most 3 files, got %d", len(files))
	}
}
