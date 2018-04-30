package circonusllhist

import (
	"bytes"
	"math"
	"testing"
)

func TestSerialize(t *testing.T) {
	h, err := NewFromStrings([]string{
		"H[0.0e+00]=1",
		"H[1.0e+01]=1",
		"H[2.0e+02]=1",
	}, false)
	if err != nil {
		t.Error("could not read from strings for test")
	}

	buf := bytes.NewBuffer([]byte{})
	if err := h.Serialize(buf); err != nil {
		t.Error(err)
	}

	h2, err := Deserialize(buf)
	if err != nil {
		t.Error(h2, err)
	}
	if !h.Equals(h2) {
		t.Log(h.DecStrings())
		t.Log(h2.DecStrings())
		t.Error("histograms do not match")
	}
}

func helpTestBin(t *testing.T, v float64, val, exp int8) {
	b := newBinFromFloat64(v)
	if b.val != val || b.exp != exp {
		t.Errorf("%v -> [%v,%v] expected, but got [%v,%v]", v, val, exp, b.val, b.exp)
	}
}

func fuzzy_equals(expected, actual float64) bool {
	delta := math.Abs(expected / 100000.0)
	if actual >= expected-delta && actual <= expected+delta {
		return true
	}
	return false
}

func TestBins(t *testing.T) {
	helpTestBin(t, 0.0, 0, 0)
	helpTestBin(t, 100, 10, 2)
	helpTestBin(t, 9.9999e-129, 0, 0)
	helpTestBin(t, 1e-128, 10, -128)
	helpTestBin(t, 1.00001e-128, 10, -128)
	helpTestBin(t, 1.09999e-128, 10, -128)
	helpTestBin(t, 1.1e-128, 11, -128)
	helpTestBin(t, 1e127, 10, 127)
	helpTestBin(t, 9.999e127, 99, 127)
	helpTestBin(t, 1e128, -1, 0)
	helpTestBin(t, -9.9999e-129, 0, 0)
	helpTestBin(t, -1e-128, -10, -128)
	helpTestBin(t, -1.00001e-128, -10, -128)
	helpTestBin(t, -1.09999e-128, -10, -128)
	helpTestBin(t, -1.1e-128, -11, -128)
	helpTestBin(t, -1e127, -10, 127)
	helpTestBin(t, -9.999e127, -99, 127)
	helpTestBin(t, -1e128, -1, 0)
	helpTestBin(t, 9.999e127, 99, 127)

	h := New()
	h.RecordIntScale(100, 1)
	if h.bvs[0].val != 10 || h.bvs[0].exp != 2 {
		t.Errorf("100 not added correctly")
	}
}

func helpTestVB(t *testing.T, v, b, w float64) {
	bin := newBinFromFloat64(v)
	out := bin.value()
	interval := bin.binWidth()
	if out < 0 {
		interval *= -1.0
	}
	if !fuzzy_equals(b, out) {
		t.Errorf("%v -> %v != %v\n", v, out, b)
	}
	if !fuzzy_equals(w, interval) {
		t.Errorf("%v -> [%v] != [%v]\n", v, interval, w)
	}
}

func TestBinSizes(t *testing.T) {
	helpTestVB(t, 43.3, 43.0, 1.0)
	helpTestVB(t, 99.9, 99.0, 1.0)
	helpTestVB(t, 10.0, 10.0, 1.0)
	helpTestVB(t, 1.0, 1.0, 0.1)
	helpTestVB(t, 0.0002, 0.0002, 0.00001)
	helpTestVB(t, 0.003, 0.003, 0.0001)
	helpTestVB(t, 0.3201, 0.32, 0.01)
	helpTestVB(t, 0.0035, 0.0035, 0.0001)
	helpTestVB(t, -1.0, -1.0, -0.1)
	helpTestVB(t, -0.00123, -0.0012, -0.0001)
	helpTestVB(t, -987324, -980000, -10000)
}
