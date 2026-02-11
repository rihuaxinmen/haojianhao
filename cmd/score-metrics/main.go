package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rihuaxinmen/haojianhao/bodymetrics"
)

func main() {
	var (
		standardsPath = flag.String("standards", "standard_body_metrics.csv", "标准体围CSV（由 standard-body-metrics 生成）")
		inPath        = flag.String("in", "measurements.csv", "个人测量值CSV")
		outPath       = flag.String("out", "scored.csv", "输出CSV")
		hasHeader     = flag.Bool("header", true, "个人测量值CSV是否包含表头（第一行）")
	)
	flag.Parse()

	std, err := bodymetrics.ReadStandardsCSV(*standardsPath)
	if err != nil {
		log.Fatal(err)
	}

	in, err := os.Open(*inPath)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	r := csv.NewReader(in)
	r.FieldsPerRecord = -1

	cols := bodymetrics.DefaultFixedColumns()
	if *hasHeader {
		header, err := r.Read()
		if err != nil {
			log.Fatal(err)
		}
		if detected, ok := bodymetrics.DetectColumns(header); ok {
			cols = detected
		}
	}

	out, err := os.Create(*outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	w := bodymetrics.NewCSVWriter(out)
	if err := w.Write(buildScoreHeader()); err != nil {
		log.Fatal(err)
	}

	okRows, badRows, noStd := 0, 0, 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			badRows++
			continue
		}

		sex, age, v, err := bodymetrics.ParseMeasurementRow(row, cols)
		if err != nil {
			badRows++
			continue
		}

		sr, ok := std.Get(sex, age)
		if !ok || sr.SampleCount <= 0 {
			noStd++
			continue
		}

		outRow := []string{itoa(sex), itoa(age), itoa(sr.SampleCount)}
		for _, name := range bodymetrics.MetricNamesExport() {
			measured := v[name]
			trip, ok := sr.V[name]
			if !ok {
				// if missing in standards, keep blanks
				outRow = append(outRow, fmt1(measured), "", "", "", "", "", "")
				continue
			}
			median, a, b := trip[0], trip[1], trip[2]
			diff := bodymetrics.SurpassDiff(measured, median)
			rp := bodymetrics.SurpassRatioPercent(measured, median)
			level := bodymetrics.StandardLevel(measured, a, b)
			outRow = append(outRow,
				fmt1(measured),
				fmt1(median),
				fmt1(diff),
				fmt1(rp),
				fmt1(a),
				fmt1(b),
				level,
			)
		}

		if err := w.Write(outRow); err != nil {
			log.Fatal(err)
		}
		okRows++
	}

	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("完成：ok=%d bad=%d no_std=%d 输出=%s\n", okRows, badRows, noStd, *outPath)
}

func buildScoreHeader() []string {
	h := []string{"sex", "age", "std_sample_count"}
	for _, n := range bodymetrics.MetricNamesExport() {
		h = append(h,
			n+"_measured",
			n+"_median", // P30
			n+"_surpass_diff",
			n+"_surpass_ratio_percent",
			n+"_a",     // P40
			n+"_b",     // P55
			n+"_level", // low|standard|high
		)
	}
	return h
}

func fmt1(v float64) string { return fmt.Sprintf("%.1f", bodymetrics.Round1(v)) }
func itoa(i int) string     { return fmt.Sprintf("%d", i) }
