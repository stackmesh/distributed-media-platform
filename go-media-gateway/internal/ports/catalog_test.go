package ports

import "testing"

func TestParseNodes(t *testing.T) {
	nodes, err := ParseNodes("us-east-1:edge-a, eu-west-1:edge-c")
	if err != nil || len(nodes) != 2 || nodes[1].ID != "edge-c" {
		t.Fatalf("%#v %v", nodes, err)
	}
	if _, err := ParseNodes("edge-a"); err == nil {
		t.Fatal("expected region:id")
	}
	nodes, err = ParseNodes("")
	if err != nil || len(nodes) != 3 {
		t.Fatalf("default %#v %v", nodes, err)
	}
}
