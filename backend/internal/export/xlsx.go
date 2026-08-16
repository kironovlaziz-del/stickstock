package export

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// ToXLSX writes into the workbook's default "Sheet1" rather than creating
// and renaming a sheet — deliberately sticking to the smallest, most
// stable slice of excelize's API (NewFile/SetCellValue/CoordinatesToCellName/
// Write) since this was written without a Go toolchain available to
// verify it compiles against the pinned excelize version.
func ToXLSX(w io.Writer, columns []string, rows [][]interface{}) error {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Sheet1"

	for i, col := range columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, col); err != nil {
			return err
		}
	}

	for r, row := range rows {
		for c, v := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return err
			}
		}
	}

	return f.Write(w)
}
