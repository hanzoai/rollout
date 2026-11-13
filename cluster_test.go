package rollout

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// PortForward must pass kubectl a ":PORT" spec — an empty local port, which
// kubectl maps to a random free port and reports on the "Forwarding from"
// line. The fake kubectl enforces the real grammar (kubectl's splitPort:
// two colon-separated parts, otherwise the spec is invalid).
func TestPortForwardUsesEmptyLocalPortSpec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake kubectl is a shell script")
	}
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "kubectl.log")
	fake := `#!/bin/sh
printf '%s ' "$@" >> "$FAKE_KUBECTL_LOG"; printf '\n' >> "$FAKE_KUBECTL_LOG"
prev=""
for a in "$@"; do
  if [ "$prev" = "get" ] && [ "$a" = "pod" ]; then echo "pod-abc"; exit 0; fi
  prev="$a"
done
state=""
for a in "$@"; do
  if [ "$a" = "port-forward" ]; then state="wantpod"; continue; fi
  if [ "$state" = "wantpod" ]; then state="wantspec"; continue; fi
  if [ "$state" = "wantspec" ]; then
    remote=$(echo "$a" | awk -F: '{if (NF==2) print $2; else print $1}')
    if [ -z "$remote" ]; then
      echo "error: remote port cannot be empty" >&2
      exit 1
    fi
    echo "Forwarding from 127.0.0.1:54321 -> $remote"
    sleep 2
    exit 0
  fi
done
echo ""
`
	path := filepath.Join(bin, "kubectl")
	if err := os.WriteFile(path, []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_KUBECTL_LOG", log)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	addr, cancel, err := PortForward(context.Background(), "kind-x", "ns", "deploy", "gateway", 8080)
	if err != nil {
		t.Fatalf("PortForward: %v", err)
	}
	defer cancel()
	if addr != "http://127.0.0.1:54321" {
		t.Errorf("addr = %q, want http://127.0.0.1:54321 (the port from the Forwarding line)", addr)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "::8080") {
		t.Errorf("kubectl received an invalid double-colon spec: %q", data)
	}
	if !strings.Contains(string(data), "port-forward pod-abc :8080") {
		t.Errorf("expected spec :8080 in kubectl args: %q", data)
	}
}

// kubectl logs a line per proxied connection to stdout. PortForward must keep
// draining the pipe after it has read the Forwarding line, or kubectl blocks on
// a full pipe. The fake writes far more than a pipe holds, then marks the file.
func TestPortForwardDrainsOutputAfterForwardingLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake kubectl is a shell script")
	}
	bin := t.TempDir()
	done := filepath.Join(t.TempDir(), "done")
	fake := `#!/bin/sh
prev=""
for a in "$@"; do
  if [ "$prev" = "get" ] && [ "$a" = "pod" ]; then echo "pod-abc"; exit 0; fi
  prev="$a"
done
echo "Forwarding from 127.0.0.1:54321 -> 8080"
i=0
while [ $i -lt 4000 ]; do
  echo "Handling connection for 54321 ........................................"
  i=$((i+1))
done
touch "$FAKE_KUBECTL_DONE"
exec sleep 5
`
	if err := os.WriteFile(filepath.Join(bin, "kubectl"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_KUBECTL_DONE", done)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, cancel, err := PortForward(context.Background(), "kind-x", "ns", "deploy", "gateway", 8080)
	if err != nil {
		t.Fatalf("PortForward: %v", err)
	}
	defer cancel()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(done); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("kubectl blocked writing to a pipe nobody drains")
}
