package bodymetrics

import "testing"

func TestQIndex(t *testing.T) {
	// n=10:
	// ceil(10*0.30)-1 = 2
	if got := qIndex(10, 0.30); got != 2 {
		t.Fatalf("qIndex(10,0.30)=%d", got)
	}
	// ceil(10*0.40)-1 = 3
	if got := qIndex(10, 0.40); got != 3 {
		t.Fatalf("qIndex(10,0.40)=%d", got)
	}
	// ceil(10*0.55)-1 = 5
	if got := qIndex(10, 0.55); got != 5 {
		t.Fatalf("qIndex(10,0.55)=%d", got)
	}
}

func TestP30MedianAB(t *testing.T) {
	arr := []float64{10, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	median, a, b := P30MedianAB(arr)
	if median != 3.0 || a != 4.0 || b != 6.0 {
		t.Fatalf("median/a/b = %.1f/%.1f/%.1f", median, a, b)
	}
}

func TestSurpass(t *testing.T) {
	if d := SurpassDiff(85, 80); d != 5.0 {
		t.Fatalf("SurpassDiff=%.1f", d)
	}
	if p := SurpassRatioPercent(85, 80); p != 6.2 {
		t.Fatalf("SurpassRatioPercent=%.1f", p)
	}
}
