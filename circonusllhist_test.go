package circonusllhist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreate(t *testing.T) {
	h := New()
	/*
		for j := 0; j < 100000; j++ {
			h.RecordIntScale(rand.Intn(1000), 0)
		}
	*/
	_ = h.RecordIntScales(99, 0, int64(rand.Intn(2))+1)
	buf := bytes.NewBuffer([]byte{})
	if err := h.Serialize(buf); err != nil {
		t.Error(err)
	}
	h2, err := Deserialize(buf)
	if err != nil {
		t.Error(err)
	}
	for j := uint16(0); j < h2.used; j++ {
		if h2.bvs[j].exp < 1 && (h2.bvs[j].val%10) != 0 {
			t.Errorf("bad bin[%v] %ve%v", j, float64(h2.bvs[j].val)/10.0, h2.bvs[j].exp)
		}
	}
}

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
	if err = h.Serialize(buf); err != nil {
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

func TestCount(t *testing.T) {
	h, err := NewFromStrings([]string{
		"H[0.0e+00]=1",
		"H[1.0e+01]=1",
		"H[2.0e+02]=1",
	}, true)
	if err != nil {
		t.Error("could not read from strings for test")
	}
	if h.Count() != 3 {
		t.Error("the count is incorrect")
	}
	err = h.RecordValue(10)
	if err != nil {
		t.Error("could not record new value to histogram")
	}
	if h.Count() != 4 {
		t.Error("the count is incorrect")
	}
}

func TestBinCount(t *testing.T) {
	h, err := NewFromStrings([]string{
		"H[0.0e+00]=1",
		"H[1.0e+01]=1",
		"H[2.0e+02]=1",
	}, true)
	if err != nil {
		t.Error("could not read from strings for test")
	}
	if h.BinCount() != 3 {
		t.Error("bin count is incorrect")
	}
}

func TestJSON(t *testing.T) {
	h, err := NewFromStrings([]string{
		"H[0.0e+00]=1",
		"H[1.0e+01]=1",
		"H[2.0e+02]=1",
	}, false)
	if err != nil {
		t.Errorf("could not read from strings for test error = %v", err)
	}

	jh, err := json.Marshal(h)
	if err != nil {
		t.Errorf("could not marshall json for test error = %v", err)
	}

	h2 := &Histogram{}
	if err := json.Unmarshal(jh, h2); err != nil {
		t.Errorf("could not unmarshall json for test error = %v", err)
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

func fuzzyEquals(expected, actual float64) bool {
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
	_ = h.RecordIntScale(100, 0)
	if h.bvs[0].val != 10 || h.bvs[0].exp != 2 {
		t.Errorf("100 not added correctly")
	}

	h = New()
	_ = h.RecordValue(100.0)
	if h.bvs[0].val != 10 || h.bvs[0].exp != 2 {
		t.Errorf("100.0 not added correctly")
	}
}

func TestRecordDuration(t *testing.T) {
	tests := []struct {
		input      []time.Duration
		inputUnit  time.Duration
		approxSum  time.Duration
		approxMean time.Duration
		tolerance  time.Duration
	}{
		{
			input:      []time.Duration{time.Nanosecond},
			approxSum:  time.Nanosecond,
			approxMean: time.Nanosecond,
		},
		{
			input:      []time.Duration{3 * time.Nanosecond},
			approxSum:  3 * time.Nanosecond,
			approxMean: 3 * time.Nanosecond,
		},
		{
			input:      []time.Duration{1000 * time.Second},
			approxSum:  1000 * time.Second,
			approxMean: 1000 * time.Second,
		},
		{
			input: []time.Duration{
				4 * time.Second,
				8 * time.Second,
			},
			approxSum:  12.0 * time.Second,
			approxMean: 6.0 * time.Second,
		},
	}

	fuzzyEquals := func(expected, actual time.Duration) bool {
		diff := math.Abs(float64(expected) - float64(actual))
		return (diff / math.Max(float64(expected), float64(actual))) <= 0.05
	}

	for n, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			h := New()
			for _, dur := range test.input {
				_ = h.RecordDuration(dur)
			}

			if v := time.Duration(1000000000.0 * h.ApproxSum()); !fuzzyEquals(v, test.approxSum) {
				t.Fatalf("%v approx sum bad: have=%v want=%v", test.input, h.ApproxSum(), test.approxSum)
			}

			if v := time.Duration(1000000000.0 * h.ApproxMean()); !fuzzyEquals(v, test.approxMean) {
				t.Fatalf("%v approx mean bad: have=%v want=%v", test.input, v, test.approxMean)
			}
		})
	}
}

func helpTestVB(t *testing.T, v, b, w float64) {
	bin := newBinFromFloat64(v)
	out := bin.value()
	interval := bin.binWidth()
	if out < 0 {
		interval *= -1.0
	}
	if !fuzzyEquals(b, out) {
		t.Errorf("%v -> %v != %v\n", v, out, b)
	}
	if !fuzzyEquals(w, interval) {
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

// preloadedTester knows how to preload values, then use them to benchmark a histogram.
type preloadedTester interface {
	preload(n int)
	run(histogram *Histogram) error
}

// intScale knows how to benchmark RecordIntScale.
type intScale struct {
	// integers hold the integers we will feed RecordIntScale
	integers []int64

	// scales hold the scales we will feed RecordIntScale
	scales []int

	// scale is the scale of the distribution of values - this allows the benchmark
	// to tease apart differences in the usage of a histogram in different applications
	// where it may be storing fairly homogenous values or any value whatsoever
	scale int

	n int
}

func (t *intScale) preload(n int) {
	t.n = 0
	t.integers = make([]int64, n)
	t.scales = make([]int, n)

	scaleMin := rand.Intn(math.MaxInt64 - t.scale)
	for i := 0; i < n; i++ {
		t.integers[i] = rand.Int63() * (rand.Int63n(2) - 1) // allow negatives!
		t.scales[i] = rand.Intn(t.scale) + scaleMin
	}
}

func (t *intScale) run(histogram *Histogram) error {
	n := t.n
	t.n++
	return histogram.RecordIntScale(t.integers[n], t.scales[n])
}

// value knows how to benchmark RecordValue.
type value struct {
	// values hold the integers we will feed RecordValue
	values []float64

	// stddev is the standard deviation of the distribution of values - this allows the
	// benchmark to tease apart differences in the usage of a histogram in different
	// applications where it may be storing fairly homogenous values or any value whatsoever
	stddev float64

	n int
}

func (t *value) preload(n int) {
	t.n = 0
	t.values = make([]float64, n)

	mean := float64(rand.Int63() * (rand.Int63n(2) - 1)) // allow negatives!
	for i := 0; i < n; i++ {
		t.values[i] = rand.NormFloat64()*t.stddev + mean
	}
}

func (t *value) run(histogram *Histogram) error {
	n := t.n
	t.n++
	return histogram.RecordValue(t.values[n])
}

func BenchmarkRecord(b *testing.B) {
	benchmarkForHist(b, func() *Histogram {
		return New()
	})
}

func BenchmarkRecordWithoutLookups(b *testing.B) {
	benchmarkForHist(b, func() *Histogram {
		return New(NoLookup())
	})
}

func benchmarkForHist(b *testing.B, constructor func() *Histogram) {
	rand.Seed(time.Now().UnixNano())
	for _, scale := range []int{1, 2, 4, 8, 16, 32, 64} {
		for _, tester := range []preloadedTester{
			&intScale{scale: scale},
			&value{stddev: math.Pow10(scale)},
		} {
			name := fmt.Sprintf("%T", tester)
			b.Run(fmt.Sprintf("%s_%d", name[strings.Index(name, ".")+1:], scale), func(b *testing.B) {
				histogram := constructor()
				tester.preload(b.N)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := tester.run(histogram); err != nil {
						b.Error(err)
					}
				}
			})
		}
	}
}

// TestCustomRoundTripping tests that clients using the HistogramWithoutLookups
// structure for custom serialization and deserialization get interchangeable
// behavior with the default spec.
func TestCustomRoundTripping(t *testing.T) {
	h := New()
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		if err := h.RecordIntScale(rand.Int63(), rand.Int()); err != nil {
			t.Fatalf("could not record numeric value: %v", err)
		}
	}

	defaultBytes, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("could not marshal histogram: %v", err)
	}

	withoutLookupBytes, err := json.Marshal(&HistogramWithoutLookups{histogram: h})
	if err != nil {
		t.Fatalf("could not marshal histogram: %v", err)
	}

	if !reflect.DeepEqual(defaultBytes, withoutLookupBytes) {
		t.Fatalf("histogram without lookups serialized into something different than default: expected %v, got %v", defaultBytes, withoutLookupBytes)
	}

	for source, data := range map[string][]byte{
		"default":        defaultBytes,
		"withoutLookups": withoutLookupBytes,
	} {
		var deserializedWithoutLookups HistogramWithoutLookups
		if err := json.Unmarshal(data, &deserializedWithoutLookups); err != nil {
			t.Fatalf("could not deserialize %s bytes into custom struct: %v", source, err)
		}
		if deserializedWithoutLookups.histogram.useLookup != false || deserializedWithoutLookups.histogram.lookup != nil {
			t.Errorf("after deserializing %s bytes into custom struct, got allocated lookup table", source)
		}
		extracted := deserializedWithoutLookups.HistogramWithLookups()
		if extracted.useLookup != true || len(extracted.lookup) != 256 {
			t.Errorf("after deserializing %s bytes into cutom struct and extracting with lookups, did not get allocated lookup table", source)
		}

		var deserializedDefault Histogram
		if err := json.Unmarshal(data, &deserializedDefault); err != nil {
			t.Fatalf("could not deserialize %s bytes into default struct: %v", source, err)
		}
		if deserializedDefault.useLookup != true || len(deserializedDefault.lookup) != 256 {
			t.Errorf("after deserializing %s bytes into default struct, did not get allocated lookup table", source)
		}
	}
}
