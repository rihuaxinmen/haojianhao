package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rihuaxinmen/haojianhao/bodymetrics"
)

func main() {
	var (
		inPath    = flag.String("in", "data.csv", "输入CSV路径（MLD VAPro-5导出）")
		outPath   = flag.String("out", "standard_body_metrics.csv", "输出CSV路径")
		hasHeader = flag.Bool("header", true, "输入CSV是否包含表头（第一行）")
	)
	flag.Parse()

	cols := bodymetrics.DefaultFixedColumns()
	pop, okRows, badRows, err := bodymetrics.ReadPopulationCSV(*inPath, cols, *hasHeader)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(*outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	w := bodymetrics.NewCSVWriter(out)
	if err := w.Write(bodymetrics.BuildHeader()); err != nil {
		log.Fatal(err)
	}

	for sex := 1; sex <= 2; sex++ {
		for age := 1; age <= 99; age++ {
			m := pop.Target(sex, age)
			if m == nil || len(m.Bust) == 0 {
				if err := w.Write(bodymetrics.BuildEmptyRow(sex, age)); err != nil {
					log.Fatal(err)
				}
				continue
			}
			if err := w.Write(bodymetrics.BuildStandardsRow(sex, age, m)); err != nil {
				log.Fatal(err)
			}
		}
	}

	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("完成：ok=%d bad=%d 输出=%s\n", okRows, badRows, *outPath)
}
