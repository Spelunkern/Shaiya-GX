// Command itempatch applies explicit CSV patches to native Item.SData without
// reserializing unrelated fields. Input and output must be different paths.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"log"
	"os"
	"strconv"
)

func main() {
	input := flag.String("input", "", "source Item.SData")
	output := flag.String("output", "", "new output file")
	source := flag.String("patches", "", "CSV: type,type_id,req_vg,stack,description")
	flag.Parse()
	if *input == "" || *output == "" || *input == *output {
		log.Fatal("distinct input and output paths required")
	}
	raw, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Open(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	patches := map[sdata.ItemKey]sdata.ItemPatch{}
	for n, row := range rows {
		if n == 0 {
			if fmt.Sprint(row) != "[type type_id req_vg stack description]" {
				log.Fatal("invalid patch header")
			}
			continue
		}
		if len(row) != 5 {
			log.Fatal("invalid patch row")
		}
		nums := [4]uint64{}
		for i := 0; i < 4; i++ {
			v, e := strconv.ParseUint(row[i], 10, 16)
			if e != nil {
				log.Fatal(e)
			}
			nums[i] = v
		}
		if nums[0] > 255 || nums[1] > 255 || nums[3] > 255 {
			log.Fatal("native byte field overflow")
		}
		patches[sdata.ItemKey{Type: uint8(nums[0]), TypeID: uint8(nums[1])}] = sdata.ItemPatch{ReqVG: uint16(nums[2]), Stack: uint8(nums[3]), Description: row[4]}
	}
	patched, err := sdata.PatchItemDefinitions(raw, patches)
	if err != nil {
		log.Fatal(err)
	}
	fout, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	if _, err = fout.Write(patched); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("patched %d native definitions\n", len(patches))
}
