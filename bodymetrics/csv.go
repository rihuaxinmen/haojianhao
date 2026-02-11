package bodymetrics

import (
	"encoding/csv"
	"io"
)

type CSVWriter struct {
	w   *csv.Writer
	err error
}

func NewCSVWriter(out io.Writer) *CSVWriter {
	return &CSVWriter{w: csv.NewWriter(out)}
}

func (cw *CSVWriter) Write(record []string) error {
	if cw.err != nil {
		return cw.err
	}
	cw.err = cw.w.Write(record)
	return cw.err
}

func (cw *CSVWriter) Flush() error {
	cw.w.Flush()
	if cw.err != nil {
		return cw.err
	}
	return cw.w.Error()
}
