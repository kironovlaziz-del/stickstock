package export

import (
	"fmt"
	"io"

	"github.com/jung-kurt/gofpdf"
)

// ToPDF renders a simple bordered table: a title, then a header row, then
// data rows, landscape A4. Column widths are evenly split — fine for a
// handful of columns, cramped for many; a real "fit to content" layout
// is a reasonable future improvement.
func ToPDF(w io.Writer, title string, columns []string, rows [][]interface{}) error {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, title)
	pdf.Ln(12)

	const pageWidth = 270.0 // usable width in mm for landscape A4 with default margins
	colWidth := pageWidth
	if len(columns) > 0 {
		colWidth = pageWidth / float64(len(columns))
	}

	pdf.SetFont("Arial", "B", 10)
	for _, col := range columns {
		pdf.CellFormat(colWidth, 8, col, "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	for _, row := range rows {
		for _, v := range row {
			pdf.CellFormat(colWidth, 7, fmt.Sprintf("%v", v), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	return pdf.Output(w)
}
