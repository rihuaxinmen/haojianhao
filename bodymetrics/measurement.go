package bodymetrics

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func ParseSex(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	switch s {
	case "男", "male", "m", "M", "1":
		return SexMale, nil
	case "女", "female", "f", "F", "2":
		return SexFemale, nil
	default:
		return 0, fmt.Errorf("invalid sex: %q", raw)
	}
}

func ParseAge(raw string) (int, error) {
	age, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || age < 1 || age > 120 {
		return 0, errors.New("invalid age")
	}
	return age, nil
}

func ParseFloat(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
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

func ColsMaxIndex(c Columns) int {
	ix := []int{
		c.Sex, c.Age,
		c.Bust, c.Waist, c.Hip,
		c.LUpperArm, c.RUpperArm,
		c.LThigh, c.RThigh,
		c.LMinThigh, c.RMinThigh,
		c.LCalf, c.RCalf,
		c.Neck, c.Height,
		c.MidWaist, c.LowWaist,
		c.Belly2Neck, c.Belly2Knee,
	}
	m := 0
	for _, v := range ix {
		if v > m {
			m = v
		}
	}
	return m
}

// ParseMeasurementRow parses one row into sex/age and metric values keyed by metricNames().
// Missing/invalid numeric values return error (caller decides skip/stop).
func ParseMeasurementRow(row []string, cols Columns) (sex, age int, v map[string]float64, err error) {
	if len(row) <= ColsMaxIndex(cols) {
		return 0, 0, nil, errors.New("not enough columns")
	}
	sex, err = ParseSex(row[cols.Sex])
	if err != nil {
		return 0, 0, nil, err
	}
	age, err = ParseAge(row[cols.Age])
	if err != nil {
		return 0, 0, nil, err
	}

	get := func(i int) (float64, error) { return ParseFloat(row[i]) }

	bust, err1 := get(cols.Bust)
	waist, err2 := get(cols.Waist)
	hip, err3 := get(cols.Hip)
	height, err4 := get(cols.Height)
	midw, err5 := get(cols.MidWaist)
	loww, err6 := get(cols.LowWaist)
	neck, err7 := get(cols.Neck)
	lua, err8 := get(cols.LUpperArm)
	rua, err9 := get(cols.RUpperArm)
	lth, err10 := get(cols.LThigh)
	rth, err11 := get(cols.RThigh)
	lminth, err12 := get(cols.LMinThigh)
	rminth, err13 := get(cols.RMinThigh)
	lcalf, err14 := get(cols.LCalf)
	rcalf, err15 := get(cols.RCalf)
	b2n, err16 := get(cols.Belly2Neck)
	b2k, err17 := get(cols.Belly2Knee)

	if err := firstErr(
		err1, err2, err3, err4, err5, err6, err7, err8, err9,
		err10, err11, err12, err13, err14, err15, err16, err17,
	); err != nil {
		return 0, 0, nil, err
	}

	v = map[string]float64{
		"bust":        bust,
		"waist":       waist,
		"hip":         hip,
		"height":      height,
		"mid_waist":   midw,
		"low_waist":   loww,
		"neck":        neck,
		"l_upper_arm": lua,
		"r_upper_arm": rua,
		"l_thigh":     lth,
		"r_thigh":     rth,
		"l_min_thigh": lminth,
		"r_min_thigh": rminth,
		"l_calf":      lcalf,
		"r_calf":      rcalf,
		"belly2neck":  b2n,
		"belly2knee":  b2k,
	}
	return sex, age, v, nil
}
