package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	sheetId        = "1WDW-gIhLJquDh_Y7LIG7Dvq75saF5SqhmGLQ9oWr_34"
	wowHeadBase    = "https://www.wowhead.com/item="
	dateFormat     = "2006-01-02"
	dateTimeFormat = "2006-01-02 15:04:05"
	overviewSheet  = "Accumulated Modifiers"
	rowDimension   = "ROWS"
)

var (
	//go:embed service-account.json
	serviceAccountJson []byte
)

type ModifierPair struct {
	Modifier int
	Player   string
}
type ModifierPairList = []ModifierPair
type ModifierMap = map[string]ModifierPairList

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
type SoftResList = []SoftRes

type Drop struct {
	Date   time.Time
	Item   string
	Winner string
	Empty  string // this is needed because the export ends in a comma for some reason
}
type DropList = []Drop

func main() {
	//for i, drop := range readLootLogCsv("data/week1-lootlog.csv") {
	//	log.Printf("[%d] %#v\n", i, drop)
	//}

	modifierMap := make(ModifierMap, 0)
	fakeList := make(ModifierPairList, 0)
	fakeList = append(fakeList, ModifierPair{
		Modifier: 20,
		Player:   "Arin",
	})
	fakeList = append(fakeList, ModifierPair{
		Modifier: 50,
		Player:   "Myrd",
	})
	fakeList = append(fakeList, ModifierPair{
		Modifier: 70,
		Player:   "Squacky",
	})
	fakeList = append(fakeList, ModifierPair{
		Modifier: 10,
		Player:   "Malv",
	})
	fakeList = append(fakeList, ModifierPair{
		Modifier: 30,
		Player:   "Info",
	})
	modifierMap["Best Item"] = fakeList
	modifierMap["Item with a really long name"] = fakeList

	fakeList2 := make(ModifierPairList, 0)
	fakeList2 = append(fakeList2, ModifierPair{
		Modifier: 60,
		Player:   "Twisd",
	})
	fakeList2 = append(fakeList2, ModifierPair{
		Modifier: 100,
		Player:   "Disarray",
	})
	modifierMap["An Item"] = fakeList2

	if err := writeModifiersToSheets(modifierMap); err != nil {
		log.Fatalf("error writing data to sheets: %v\n", err)
	}

	log.Println("successfully wrote data to sheets")
}

func readLootLogCsv(filename string) DropList {
	dropList := make([]Drop, 0)

	readAndProcessCsv(filename, false, func(record []string) {
		d, _ := time.Parse(dateFormat, record[0])

		dropList = append(dropList, Drop{
			Date:   d,
			Item:   record[1],
			Winner: record[2],
			Empty:  record[3],
		})
	})

	return dropList
}

func readSoftResCsv(filename string) SoftResList {
	resList := make([]SoftRes, 0)

	readAndProcessCsv(filename, true, func(record []string) {
		mod, _ := strconv.Atoi(record[7])
		d, _ := time.Parse(dateTimeFormat, record[8])

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
	})

	return resList
}

func readAndProcessCsv(filename string, hasHeader bool, processor func(record []string)) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln("failed to open soft reserve file", err)
	}
	defer f.Close()

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
		if hasHeader && header {
			header = false
			continue
		}

		processor(record)
	}
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

func writeModifiersToSheets(modifierMap ModifierMap) error {
	itemList := make([]string, 0)
	for key := range modifierMap {
		itemList = append(itemList, key)
	}
	sort.Strings(itemList)

	values := make([][]interface{}, 0)

	for _, key := range itemList {
		row := make([]interface{}, 2)
		row[0] = key
		row[1] = formatModifiers(modifierMap[key])
		values = append(values, row)
	}

	n := strconv.Itoa(len(itemList) + 1)

	data := []*sheets.ValueRange{{
		MajorDimension: rowDimension,
		Range:          overviewSheet + "!A2" + ":B" + n,
		Values:         values,
	}}

	valueInputOption := "USER_ENTERED"

	rb := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: valueInputOption,
		Data:             data,
	}

	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(serviceAccountJson))
	if err != nil {
		log.Fatalf("Unable to get sheets data: %v\n", err)
	}
	_, err = srv.Spreadsheets.Values.BatchUpdate(sheetId, rb).Context(ctx).Do()
	if err != nil {
		log.Fatalf("failed to update modifier sheet! %v\n", err)
	}

	log.Println("updated maybe. go check!")

	return nil
}

func formatModifiers(modifiers ModifierPairList) string {
	sort.Slice(modifiers, func(i, j int) bool {
		return modifiers[i].Modifier > modifiers[j].Modifier
	})

	s := ""

	for _, mod := range modifiers {
		s += fmt.Sprintf("%d: %s\n", mod.Modifier, mod.Player)
	}

	return s
}

func getItemLink(itemId string) string {
	return wowHeadBase + itemId
}
