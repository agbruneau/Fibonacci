package system

import "testing"

func TestSample_ReturnsValidRanges(t *testing.T) {
	s := Sample()
	if s.CPUPercent < 0 || s.CPUPercent > 100 {
		t.Errorf("CPUPercent out of range: %f", s.CPUPercent)
	}
	if s.MemPercent < 0 || s.MemPercent > 100 {
		t.Errorf("MemPercent out of range: %f", s.MemPercent)
	}
}

func TestSample_MemPercentNonZero(t *testing.T) {
	s := Sample()
	if s.MemPercent == 0 {
		t.Error("expected non-zero MemPercent on a running system")
	}
}
