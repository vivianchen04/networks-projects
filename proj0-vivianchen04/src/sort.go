package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"sort"
)

func readInputFile(filename string) [][]byte {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal("Reading file error", err)
	}

	defer f.Close()

	// save record into recordAll
	recordAll := make([][]byte, 0) // not sure

	for {
		record := make([]byte, 100)
		_, err := f.Read(record) // read up to len(buf) bytes, maybe lesser
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		if err == io.EOF {
			break
		}

		recordAll = append(recordAll, record)
	}

	return recordAll
}

func writeOutputFile(filename string, recordAll [][]byte) {
	// sort recordAll
	sort.Slice(recordAll, func(p, q int) bool {
		return bytes.Compare(recordAll[p][:10], recordAll[q][:10]) == -1
	})

	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for _, record := range recordAll {
		f.Write(record)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) != 3 {
		log.Fatalf("Usage: %v inputfile outputfile\n", os.Args[0])
	}

	recordAll := readInputFile(os.Args[1])

	log.Printf("Sorting %s to %s\n", os.Args[1], os.Args[2])

	writeOutputFile(os.Args[2], recordAll)

}
