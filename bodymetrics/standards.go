package bodymetrics

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Standards struct {
	Rows map[int]map[int]StandardsRow // sex->age->row
}

func ReadStandardsCSV(path string) (*Standards, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}

	req := []string{"sex", "age", "sample_count"}
	for _, k := range req {
		if _, ok := col[k]; !ok {
			return nil, fmt.Errorf("missing column %q", k)
		}
	}

	s := &Standards{Rows: map[int]map[int]StandardsRow{}}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// keep going on bad lines
			continue
		}
		sex, err1 := strconv.Atoi(strings.TrimSpace(row[col["sex"]]))
		age, err2 := strconv.Atoi(strings.TrimSpace(row[col["age"]]))
		sc, err3 := strconv.Atoi(strings.TrimSpace(row[col["sample_count"]]))
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		sr := StandardsRow{Sex: sex, Age: age, SampleCount: sc, V: map[string][3]float64{}}
		for _, name := range metricNames() {
			mk := name + "_median"
			ak := name + "_a"
			bk := name + "_b"
			mi, ok1 := col[mk]
			ai, ok2 := col[ak]
			bi, ok3 := col[bk]
			if !ok1 || !ok2 || !ok3 {
				continue
			}
			if mi >= len(row) || ai >= len(row) || bi >= len(row) {
				continue
			}
			median, e1 := strconv.ParseFloat(strings.TrimSpace(row[mi]), 64)
			a, e2 := strconv.ParseFloat(strings.TrimSpace(row[ai]), 64)
			b, e3 := strconv.ParseFloat(strings.TrimSpace(row[bi]), 64)
			if e1 != nil || e2 != nil || e3 != nil {
				continue
			}
			sr.V[name] = [3]float64{median, a, b}
		}

		if _, ok := s.Rows[sex]; !ok {
			s.Rows[sex] = map[int]StandardsRow{}
		}
		s.Rows[sex][age] = sr
	}

	return s, nil
}

func (s *Standards) Get(sex, age int) (StandardsRow, bool) {
	if s == nil || s.Rows == nil || s.Rows[sex] == nil {
		return StandardsRow{}, false
	}
	sr, ok := s.Rows[sex][age]
	return sr, ok
}
