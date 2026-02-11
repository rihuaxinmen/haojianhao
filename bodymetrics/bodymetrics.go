package bodymetrics

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Sex: 1=male, 2=female
const (
	SexMale   = 1
	SexFemale = 2
)

type Metrics struct {
	Bust, Waist, Hip       []float64
	LUpperArm, RUpperArm   []float64
	LThigh, RThigh         []float64
	LMinThigh, RMinThigh   []float64
	LCalf, RCalf           []float64
	Neck                   []float64
	MidWaist, LowWaist     []float64
	Height                 []float64
	Belly2Neck, Belly2Knee []float64
}

type StandardsRow struct {
	Sex         int
	Age         int
	SampleCount int
	// key: metric name, value: [median(P30), a(P40), b(P55)]
	V map[string][3]float64
}

// Columns defines CSV column indices for reading.
// If DetectColumns fails, you can fill these with fixed indices.
type Columns struct {
	Sex, Age int

	Bust, Waist, Hip               int
	LUpperArm, RUpperArm           int
	LThigh, RThigh                 int
	LMinThigh, RMinThigh           int
	LCalf, RCalf                   int
	Neck, MidWaist, LowWaist       int
	Height, Belly2Neck, Belly2Knee int
}

// DetectColumns tries to detect a known export header.
// Returns ok=false if not confidently detected.
func DetectColumns(header []string) (cols Columns, ok bool) {
	norm := func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.ReplaceAll(s, "（", "(")
		s = strings.ReplaceAll(s, "）", ")")
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, " ", "")
		return s
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[norm(h)] = i
	}
	getAny := func(keys ...string) (int, bool) {
		for _, k := range keys {
			if v, ok := idx[norm(k)]; ok {
				return v, true
			}
		}
		return 0, false
	}

	var found int
	set := func(dst *int, keys ...string) bool {
		i, ok := getAny(keys...)
		if !ok {
			return false
		}
		*dst = i
		found++
		return true
	}

	okSex := set(&cols.Sex, "sex", "性别")
	okAge := set(&cols.Age, "age", "年龄", "周岁")

	// Common Chinese exports (best-effort).
	_ = set(&cols.Bust, "bust", "胸围")
	_ = set(&cols.Waist, "waist", "腰围")
	_ = set(&cols.Hip, "hip", "臀围")
	_ = set(&cols.Height, "height", "身高")
	_ = set(&cols.MidWaist, "midwaist", "mid_waist", "中腰围", "中腰")
	_ = set(&cols.LowWaist, "lowwaist", "low_waist", "低腰围", "低腰")
	_ = set(&cols.Neck, "neck", "颈围", "脖围")
	_ = set(&cols.LUpperArm, "lupperarm", "l_upper_arm", "左上臂围", "左上臂")
	_ = set(&cols.RUpperArm, "rupperarm", "r_upper_arm", "右上臂围", "右上臂")
	_ = set(&cols.LThigh, "lthigh", "l_thigh", "左大腿围", "左大腿")
	_ = set(&cols.RThigh, "rthigh", "r_thigh", "右大腿围", "右大腿")
	_ = set(&cols.LMinThigh, "lminthigh", "l_min_thigh", "左最小大腿围", "左小腿上围", "左大腿最小围")
	_ = set(&cols.RMinThigh, "rminthigh", "r_min_thigh", "右最小大腿围", "右小腿上围", "右大腿最小围")
	_ = set(&cols.LCalf, "lcalf", "l_calf", "左小腿围", "左腓围")
	_ = set(&cols.RCalf, "rcalf", "r_calf", "右小腿围", "右腓围")
	_ = set(&cols.Belly2Neck, "belly2neck", "belly_to_neck", "腹部到颈点", "腹到颈")
	_ = set(&cols.Belly2Knee, "belly2knee", "belly_to_knee", "腹部到膝点", "腹到膝")

	// We require at least sex+age to be meaningful.
	if !okSex || !okAge {
		return Columns{}, false
	}
	// Require some body metrics too; otherwise detection is too weak.
	if found < 6 {
		return Columns{}, false
	}
	return cols, true
}

func DefaultFixedColumns() Columns {
	// Matches the user's original snippet (0-based):
	// row[1]=sex(男/女), row[2]=age, row[4..20] metrics.
	return Columns{
		Sex: 1,
		Age: 2,

		Bust:       4,
		Waist:      5,
		Hip:        6,
		LUpperArm:  7,
		RUpperArm:  8,
		LThigh:     9,
		RThigh:     10,
		LCalf:      11,
		RCalf:      12,
		Belly2Neck: 13,
		Belly2Knee: 14,
		Neck:       15,
		Height:     16,
		LMinThigh:  17,
		RMinThigh:  18,
		MidWaist:   19,
		LowWaist:   20,
	}
}

type Population struct {
	// group[sex][age] => metrics
	Group map[int]map[int]*Metrics

	MaleElder   *Metrics // 63-99 merged
	FemaleElder *Metrics // 70-99 merged
}

func NewPopulation() *Population {
	return &Population{
		Group:       map[int]map[int]*Metrics{},
		MaleElder:   &Metrics{},
		FemaleElder: &Metrics{},
	}
}

func (p *Population) Add(sex, age int, m *Metrics) {
	if _, ok := p.Group[sex]; !ok {
		p.Group[sex] = map[int]*Metrics{}
	}
	if _, ok := p.Group[sex][age]; !ok {
		p.Group[sex][age] = &Metrics{}
	}
	appendMetrics(p.Group[sex][age], m)

	if sex == SexMale && age >= 63 && age <= 99 {
		appendMetrics(p.MaleElder, m)
	}
	if sex == SexFemale && age >= 70 && age <= 99 {
		appendMetrics(p.FemaleElder, m)
	}
}

func (p *Population) Target(sex, age int) *Metrics {
	if sex == SexMale && age >= 63 && age <= 99 {
		return p.MaleElder
	}
	if sex == SexFemale && age >= 70 && age <= 99 {
		return p.FemaleElder
	}
	if p.Group[sex] == nil {
		return nil
	}
	return p.Group[sex][age]
}

func ReadPopulationCSV(path string, cols Columns, hasHeader bool) (*Population, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	if hasHeader {
		header, err := r.Read()
		if err != nil {
			return nil, 0, 0, fmt.Errorf("read header: %w", err)
		}
		if detected, ok := DetectColumns(header); ok {
			cols = detected
		}
	}

	maxIdx := func() int {
		ix := []int{
			cols.Sex, cols.Age,
			cols.Bust, cols.Waist, cols.Hip,
			cols.LUpperArm, cols.RUpperArm,
			cols.LThigh, cols.RThigh,
			cols.LMinThigh, cols.RMinThigh,
			cols.LCalf, cols.RCalf,
			cols.Neck, cols.Height,
			cols.MidWaist, cols.LowWaist,
			cols.Belly2Neck, cols.Belly2Knee,
		}
		m := 0
		for _, v := range ix {
			if v > m {
				m = v
			}
		}
		return m
	}()

	pop := NewPopulation()
	okRows, badRows := 0, 0

	parseFloat := func(s string) (float64, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, errors.New("empty")
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, errors.New("nan/inf")
		}
		return v, nil
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			badRows++
			continue
		}
		if len(row) <= maxIdx {
			badRows++
			continue
		}

		sexRaw := strings.TrimSpace(row[cols.Sex])
		sex := SexFemale
		switch sexRaw {
		case "男", "male", "m", "M", "1":
			sex = SexMale
		case "女", "female", "f", "F", "2":
			sex = SexFemale
		default:
			// default to female is dangerous; treat as invalid.
			badRows++
			continue
		}

		age, err := strconv.Atoi(strings.TrimSpace(row[cols.Age]))
		if err != nil || age < 1 || age > 120 {
			badRows++
			continue
		}

		get := func(i int) (float64, error) { return parseFloat(row[i]) }

		bust, err1 := get(cols.Bust)
		waist, err2 := get(cols.Waist)
		hip, err3 := get(cols.Hip)
		lua, err4 := get(cols.LUpperArm)
		rua, err5 := get(cols.RUpperArm)
		lth, err6 := get(cols.LThigh)
		rth, err7 := get(cols.RThigh)
		lcalf, err8 := get(cols.LCalf)
		rcalf, err9 := get(cols.RCalf)
		b2n, err10 := get(cols.Belly2Neck)
		b2k, err11 := get(cols.Belly2Knee)
		neck, err12 := get(cols.Neck)
		height, err13 := get(cols.Height)
		lminth, err14 := get(cols.LMinThigh)
		rminth, err15 := get(cols.RMinThigh)
		midw, err16 := get(cols.MidWaist)
		loww, err17 := get(cols.LowWaist)

		if err := firstErr(
			err1, err2, err3, err4, err5, err6, err7, err8, err9,
			err10, err11, err12, err13, err14, err15, err16, err17,
		); err != nil {
			badRows++
			continue
		}

		m := &Metrics{
			Bust:       []float64{bust},
			Waist:      []float64{waist},
			Hip:        []float64{hip},
			LUpperArm:  []float64{lua},
			RUpperArm:  []float64{rua},
			LThigh:     []float64{lth},
			RThigh:     []float64{rth},
			LCalf:      []float64{lcalf},
			RCalf:      []float64{rcalf},
			Belly2Neck: []float64{b2n},
			Belly2Knee: []float64{b2k},
			Neck:       []float64{neck},
			Height:     []float64{height},
			LMinThigh:  []float64{lminth},
			RMinThigh:  []float64{rminth},
			MidWaist:   []float64{midw},
			LowWaist:   []float64{loww},
		}

		pop.Add(sex, age, m)
		okRows++
	}

	return pop, okRows, badRows, nil
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func appendMetrics(dst, src *Metrics) {
	dst.Bust = append(dst.Bust, src.Bust...)
	dst.Waist = append(dst.Waist, src.Waist...)
	dst.Hip = append(dst.Hip, src.Hip...)
	dst.LUpperArm = append(dst.LUpperArm, src.LUpperArm...)
	dst.RUpperArm = append(dst.RUpperArm, src.RUpperArm...)
	dst.LThigh = append(dst.LThigh, src.LThigh...)
	dst.RThigh = append(dst.RThigh, src.RThigh...)
	dst.LMinThigh = append(dst.LMinThigh, src.LMinThigh...)
	dst.RMinThigh = append(dst.RMinThigh, src.RMinThigh...)
	dst.LCalf = append(dst.LCalf, src.LCalf...)
	dst.RCalf = append(dst.RCalf, src.RCalf...)
	dst.Neck = append(dst.Neck, src.Neck...)
	dst.Height = append(dst.Height, src.Height...)
	dst.MidWaist = append(dst.MidWaist, src.MidWaist...)
	dst.LowWaist = append(dst.LowWaist, src.LowWaist...)
	dst.Belly2Neck = append(dst.Belly2Neck, src.Belly2Neck...)
	dst.Belly2Knee = append(dst.Belly2Knee, src.Belly2Knee...)
}

func metricNames() []string {
	return []string{
		"bust", "waist", "hip", "height", "mid_waist", "low_waist", "neck",
		"l_upper_arm", "r_upper_arm",
		"l_thigh", "r_thigh",
		"l_min_thigh", "r_min_thigh",
		"l_calf", "r_calf",
		"belly2neck", "belly2knee",
	}
}

// MetricNamesExport returns the metrics in a stable order for CSV export.
func MetricNamesExport() []string {
	return append([]string(nil), metricNames()...)
}

func BuildHeader() []string {
	h := []string{"sex", "age", "sample_count"}
	for _, n := range metricNames() {
		h = append(h, n+"_median", n+"_a", n+"_b")
	}
	return h
}

func BuildEmptyRow(sex, age int) []string {
	row := []string{strconv.Itoa(sex), strconv.Itoa(age), "0"}
	// 17 metrics * 3 columns
	for i := 0; i < 51; i++ {
		row = append(row, "0")
	}
	return row
}

func Round1(v float64) float64 { return math.Round(v*10) / 10 }

func qIndex(n int, p float64) int {
	if n <= 0 {
		return 0
	}
	idx := int(math.Ceil(float64(n)*p)) - 1
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

// P30MedianAB returns (median=P30, a=P40, b=P55) per spec, all rounded to 1 decimal.
// Note: this sorts arr in place.
func P30MedianAB(arr []float64) (median, a, b float64) {
	sort.Float64s(arr)
	n := len(arr)
	median = Round1(arr[qIndex(n, 0.30)])
	a = Round1(arr[qIndex(n, 0.40)])
	b = Round1(arr[qIndex(n, 0.55)])
	return
}

func BuildStandardsRow(sex, age int, m *Metrics) []string {
	row := []string{strconv.Itoa(sex), strconv.Itoa(age), strconv.Itoa(len(m.Bust))}

	add := func(arr []float64) {
		median, a, b := P30MedianAB(arr)
		row = append(row,
			fmt.Sprintf("%.1f", median),
			fmt.Sprintf("%.1f", a),
			fmt.Sprintf("%.1f", b),
		)
	}

	add(append([]float64(nil), m.Bust...))
	add(append([]float64(nil), m.Waist...))
	add(append([]float64(nil), m.Hip...))
	add(append([]float64(nil), m.Height...))
	add(append([]float64(nil), m.MidWaist...))
	add(append([]float64(nil), m.LowWaist...))
	add(append([]float64(nil), m.Neck...))
	add(append([]float64(nil), m.LUpperArm...))
	add(append([]float64(nil), m.RUpperArm...))
	add(append([]float64(nil), m.LThigh...))
	add(append([]float64(nil), m.RThigh...))
	add(append([]float64(nil), m.LMinThigh...))
	add(append([]float64(nil), m.RMinThigh...))
	add(append([]float64(nil), m.LCalf...))
	add(append([]float64(nil), m.RCalf...))
	add(append([]float64(nil), m.Belly2Neck...))
	add(append([]float64(nil), m.Belly2Knee...))
	return row
}

// StandardLevel returns "low" | "standard" | "high" by a/b range.
func StandardLevel(measured, a, b float64) string {
	if measured < a {
		return "low"
	}
	if measured > b {
		return "high"
	}
	return "standard"
}

// SurpassDiff is "measurement - median(P30)" per spec.
func SurpassDiff(measured, median float64) float64 { return Round1(measured - median) }

// SurpassRatioPercent is (measurement-median)/median * 100.
// If median==0, returns 0.
func SurpassRatioPercent(measured, median float64) float64 {
	if median == 0 {
		return 0
	}
	return Round1((measured - median) / median * 100.0)
}
