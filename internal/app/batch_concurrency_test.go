package app

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/failure"
)

func TestNormalizeJobs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		requested  int
		candidates int
		want       int
		wantErr    bool
	}{
		{name: "unset means sequential", requested: 0, candidates: 10, want: 1},
		{name: "explicit one", requested: 1, candidates: 10, want: 1},
		{name: "below the candidate count", requested: 4, candidates: 10, want: 4},
		{name: "capped at the candidate count", requested: 16, candidates: 3, want: 3},
		{name: "equal to the candidate count", requested: 5, candidates: 5, want: 5},
		{name: "negative is rejected", requested: -1, candidates: 10, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeJobs(test.requested, test.candidates)
			if test.wantErr {
				assertFailureKind(t, err, failure.Usage)
				return
			}
			if err != nil {
				t.Fatalf("normalizeJobs() error = %v", err)
			}
			if got != test.want {
				t.Errorf("normalizeJobs(%d, %d) = %d, want %d", test.requested, test.candidates, got, test.want)
			}
		})
	}
}

func TestBatchConvertRejectsNegativeJobs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.webm"))

	_, err := New(missingBinaries(t)).BatchConvert(context.Background(), BatchRequest{
		InputDir: root,
		Target:   "mp4",
		Jobs:     -2,
	})
	assertFailureKind(t, err, failure.Usage)
}

// The whole point of collecting by index instead of by completion is that a
// concurrent run reports exactly what a sequential one would. If this breaks,
// --json output becomes nondeterministic and anything parsing it drifts.
func TestBatchConvertReportsTheSameResultAtEveryConcurrency(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Names that sort differently from the order short conversions would
	// plausibly finish in.
	for _, name := range []string{"zulu.webm", "alpha.mkv", "mike.mov", "bravo.avi", "yankee.m4v", "charlie.qt"} {
		writeTestFile(t, filepath.Join(root, name))
	}

	wantOrder := []string{
		filepath.Join(root, "alpha.mkv"),
		filepath.Join(root, "bravo.avi"),
		filepath.Join(root, "charlie.qt"),
		filepath.Join(root, "mike.mov"),
		filepath.Join(root, "yankee.m4v"),
		filepath.Join(root, "zulu.webm"),
	}

	service := New(missingBinaries(t))
	// One output directory for every run: the comparison below includes
	// OutputPath, so a fresh directory per run would differ for the wrong
	// reason. Nothing is written, because every conversion fails first.
	outputDir := t.TempDir()
	var reference []BatchItem

	for _, jobs := range []int{1, 2, 3, 6, 32} {
		result, err := service.BatchConvert(context.Background(), BatchRequest{
			InputDir:  root,
			OutputDir: outputDir,
			Target:    "mp4",
			Jobs:      jobs,
		})
		if err != nil {
			t.Fatalf("BatchConvert(jobs=%d) error = %v", jobs, err)
		}

		order := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			order = append(order, item.InputPath)
		}
		if !reflect.DeepEqual(order, wantOrder) {
			t.Errorf("jobs=%d reported items out of candidate order:\n got: %v\nwant: %v", jobs, order, wantOrder)
		}
		if result.Total != len(wantOrder) {
			t.Errorf("jobs=%d Total = %d, want %d", jobs, result.Total, len(wantOrder))
		}
		if result.Converted+result.Failed != result.Total {
			t.Errorf("jobs=%d counted %d converted + %d failed, want %d", jobs, result.Converted, result.Failed, result.Total)
		}

		if reference == nil {
			reference = result.Items
			continue
		}
		if !reflect.DeepEqual(result.Items, reference) {
			t.Errorf("jobs=%d produced different items than the sequential run", jobs)
		}
	}
}

func TestBatchConvertReportsTheResolvedJobCount(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.webm"))
	writeTestFile(t, filepath.Join(root, "b.mov"))

	service := New(missingBinaries(t))

	tests := []struct {
		name      string
		requested int
		want      int
	}{
		{name: "unset resolves to one", requested: 0, want: 1},
		{name: "kept when it fits the work", requested: 2, want: 2},
		{name: "capped at the file count", requested: 64, want: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := service.BatchConvert(context.Background(), BatchRequest{
				InputDir:  root,
				OutputDir: t.TempDir(),
				Target:    "mp4",
				Jobs:      test.requested,
			})
			if err != nil {
				t.Fatalf("BatchConvert() error = %v", err)
			}
			if result.Jobs != test.want {
				t.Errorf("Jobs = %d, want %d", result.Jobs, test.want)
			}
		})
	}
}

func TestBatchConvertStopsOnCancellationWhileConcurrent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, name := range []string{"a.webm", "b.webm", "c.webm", "d.webm"} {
		writeTestFile(t, filepath.Join(root, name))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := New(missingBinaries(t)).BatchConvert(ctx, BatchRequest{
		InputDir:  root,
		OutputDir: t.TempDir(),
		Target:    "mp4",
		Jobs:      4,
	})

	assertFailureKind(t, err, failure.Interrupted)
	if len(result.Items) != 0 {
		t.Errorf("Items = %d, want nothing processed after cancellation", len(result.Items))
	}
	if result.Converted != 0 || result.Failed != 0 {
		t.Errorf("counted %d converted and %d failed after cancellation", result.Converted, result.Failed)
	}
}
