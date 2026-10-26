package domain

import "testing"

func TestStringListRoundTrip(t *testing.T) {
	value, err := StringList{"Go", "needs \"clearance\""}.Value()
	if err != nil {
		t.Fatal(err)
	}

	var scanned StringList
	if err := scanned.Scan(value); err != nil {
		t.Fatal(err)
	}
	if len(scanned) != 2 || scanned[1] != `needs "clearance"` {
		t.Errorf("round trip lost data: %q", scanned)
	}

	// Rows from before the fit columns existed hold NULL.
	var empty StringList
	if err := empty.Scan(nil); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("NULL should scan to an empty list, got %#v (%v)", empty, err)
	}
	if nilValue, _ := StringList(nil).Value(); nilValue != "[]" {
		t.Errorf("nil list should store [], got %v", nilValue)
	}
}
