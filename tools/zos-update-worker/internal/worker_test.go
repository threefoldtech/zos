package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	client "github.com/threefoldtech/substrate-client"
)

func newTestWorker(t *testing.T) *Worker {
	t.Helper()
	testDir := t.TempDir()
	src := filepath.Join(testDir, "tf-autobuilder")
	dst := filepath.Join(testDir, "tf-zos")

	if err := os.Mkdir(src, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dst, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	worker, err := NewWorker(src, dst, Params{Interval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

func TestApplyVersion(t *testing.T) {
	// During a canary (safe_to_upgrade == false) the latest link must NOT be advanced,
	// even if the source flist exists, so new/non-canary nodes stay on GA.
	t.Run("gated when not safe to upgrade", func(t *testing.T) {
		worker := newTestWorker(t)
		const version = "v3.1.1"
		if err := os.Mkdir(filepath.Join(worker.src, ".tag-"+version), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if err := worker.applyVersion(MainNetwork, ChainVersion{Version: version, SafeToUpgrade: false}); err != nil {
			t.Fatalf("expected no error when gated, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(worker.dst, string(MainNetwork))); !os.IsNotExist(err) {
			t.Fatalf("expected no latest link to be created while gated")
		}
	})

	// safe_to_upgrade but the source flist is missing: the link update must fail.
	t.Run("fails when safe but source missing", func(t *testing.T) {
		worker := newTestWorker(t)
		if err := worker.applyVersion(MainNetwork, ChainVersion{Version: "v3.1.1", SafeToUpgrade: true}); err == nil {
			t.Fatalf("expected error when the source flist is missing")
		}
	})

	// safe_to_upgrade and the source exists: the latest link is created pointing at it.
	t.Run("links latest when safe and source exists", func(t *testing.T) {
		worker := newTestWorker(t)
		const version = "v3.1.1"
		if err := os.Mkdir(filepath.Join(worker.src, ".tag-"+version), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if err := worker.applyVersion(MainNetwork, ChainVersion{Version: version, SafeToUpgrade: true}); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		target, err := os.Readlink(filepath.Join(worker.dst, string(MainNetwork)))
		if err != nil {
			t.Fatalf("expected latest link to be created: %v", err)
		}
		if filepath.Base(target) != ".tag-"+version {
			t.Fatalf("expected link to point at .tag-%s, got %s", version, target)
		}
	})
}

// TestUpdateZosVersionUnreachable checks the connect/fetch path reports an error when the
// substrate endpoint is unreachable.
func TestUpdateZosVersionUnreachable(t *testing.T) {
	worker := newTestWorker(t)
	worker.substrate[QANetwork] = client.NewManager("wss://tfchain.qa1.grid.tf/ws")

	if err := worker.updateZosVersion(QANetwork, worker.substrate[QANetwork]); err == nil {
		t.Fatalf("expected updateZosVersion to fail against an unreachable substrate url")
	}
}
