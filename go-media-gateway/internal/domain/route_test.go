package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestRoutePrefersHomeAndIsStable(t *testing.T) {
	nodes := []Node{
		{ID: "edge-c", Region: "eu-west-1"},
		{ID: "edge-b", Region: "us-east-1"},
		{ID: "edge-a", Region: "us-east-1"},
	}
	in := RouteInput{
		ContentHash: strings.Repeat("ab", 32),
		SizeBytes:   1024,
		MIME:        "video/mp4",
		Region:      "us-east-1",
	}
	policy := Policy{MaxBytes: 1 << 20, Replicas: 2}

	first, err := Route(nodes, in, policy)
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := Route([]Node{nodes[2], nodes[1], nodes[0]}, in, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, reversed) {
		t.Fatalf("placement changed with input order: %#v vs %#v", first, reversed)
	}
	if first.Region != "us-east-1" || first.Primary == "edge-c" {
		t.Fatalf("home region lost: %#v", first)
	}
	if len(first.Replicas) != 1 || first.Replicas[0] == first.Primary {
		t.Fatalf("replica set: %#v", first)
	}
	if first.Replicas[0] == "edge-c" {
		t.Fatal("home region should fill the replica set before spill")
	}
}

func TestRouteSpillsWhenHomeIsShort(t *testing.T) {
	nodes := []Node{
		{ID: "edge-c", Region: "eu-west-1"},
		{ID: "edge-a", Region: "us-east-1"},
	}
	got, err := Route(nodes, RouteInput{
		ContentHash: strings.Repeat("cd", 32),
		SizeBytes:   8,
		MIME:        "audio/mpeg",
		Region:      "us-east-1",
	}, Policy{MaxBytes: 100, Replicas: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Primary != "edge-a" || len(got.Replicas) != 1 || got.Replicas[0] != "edge-c" {
		t.Fatalf("spill placement: %#v", got)
	}
}

func TestRouteRejectsPolicyViolations(t *testing.T) {
	nodes := []Node{{ID: "edge-a", Region: "us-east-1"}}
	base := RouteInput{ContentHash: strings.Repeat("aa", 32), SizeBytes: 4, MIME: "image/png", Region: "us-east-1"}
	policy := Policy{MaxBytes: 4, Replicas: 1}

	if _, err := Route(nodes, base, policy); err != nil {
		t.Fatal(err)
	}
	tooBig := base
	tooBig.SizeBytes = 5
	if _, err := Route(nodes, tooBig, policy); err != ErrTooLarge {
		t.Fatalf("size: %v", err)
	}
	badMIME := base
	badMIME.MIME = "text/plain"
	if _, err := Route(nodes, badMIME, policy); err != ErrMIME {
		t.Fatalf("mime: %v", err)
	}
	if _, err := Route(nil, base, policy); err != ErrNoNodes {
		t.Fatalf("nodes: %v", err)
	}
}

func TestNormalizeMIMEAndTenant(t *testing.T) {
	mime, err := NormalizeMIME("Video/MP4; codecs=avc1")
	if err != nil || mime != "video/mp4" {
		t.Fatalf("mime %q %v", mime, err)
	}
	if _, err := NormalizeMIME("application/octet-stream"); err != ErrMIME {
		t.Fatalf("expected mime rejection, got %v", err)
	}
	if !ValidTenant("demo") || !ValidTenant("a") || !ValidTenant("studio-east") {
		t.Fatal("expected valid tenants")
	}
	if ValidTenant("") || ValidTenant("-demo") || ValidTenant("Demo") || ValidTenant("demo-") {
		t.Fatal("expected invalid tenants")
	}
	if AssetID("demo", strings.Repeat("ab", 32)) != AssetID("demo", strings.Repeat("ab", 32)) {
		t.Fatal("asset id drifted")
	}
}
