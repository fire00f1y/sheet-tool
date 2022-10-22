package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	sheetId = "1WDW-gIhLJquDh_Y7LIG7Dvq75saF5SqhmGLQ9oWr_34"
)

var (
	//go:embed service-account.json
	serviceAccountJson []byte
)

type SoftRes struct {
	Item     string
	ItemId   string
	Boss     string
	Player   string
	Class    string
	Spec     string
	Note     string
	Modifier int
	Date     time.Time
}

type Drop struct {
	Date   time.Time
	Item   string
	Winner string
	Empty  string // this is needed because the export ends in a comma for some reason
}

func main() {
	for i, drop := range readLootLogCsv("data/week1-lootlog.csv") {
		log.Printf("[%d] %#v\n", i, drop)
	}
}

func readLootLogCsv(filename string) []Drop {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln("failed to open loot log file", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	dropList := make([]Drop, 0)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error reading line in loot log: %v\n", err)
			continue
		}
		d, _ := time.Parse("2006-01-02", record[0])

		dropList = append(dropList, Drop{
			Date:   d,
			Item:   record[1],
			Winner: record[2],
			Empty:  record[3],
		})
	}

	return dropList
}

func readSoftResCsv(filename string) []SoftRes {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln("failed to open soft reserve file", err)
	}
	defer f.Close()

	resList := make([]SoftRes, 0)

	r := csv.NewReader(f)
	header := true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading csv: %v\n", err)
			continue
		}
		if header {
			header = false
			continue
		}

		mod, _ := strconv.Atoi(record[7])
		d, _ := time.Parse("2006-01-02 15:04:05", record[8])

		resList = append(resList, SoftRes{
			Item:     record[0],
			ItemId:   record[1],
			Boss:     record[2],
			Player:   record[3],
			Class:    record[4],
			Spec:     record[5],
			Note:     record[6],
			Modifier: mod,
			Date:     d,
		})
	}

	return resList
}

func readFromSheets() {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(serviceAccountJson))
	if err != nil {
		log.Fatalf("Unable to get sheets data: %v\n", err)
	}
	spreadsheet, err := srv.Spreadsheets.Get(sheetId).Context(ctx).IncludeGridData(true).Do()
	if err != nil {
		log.Fatalf("error fetching the spreadsheet: %v\n", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		log.Printf("%s: %s :: (%d, %d)\n", sheet.Properties.Title, sheet.Properties.SheetType, sheet.Properties.GridProperties.RowCount, sheet.Properties.GridProperties.ColumnCount)
	}
}
