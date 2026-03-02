package export

import (
	"fmt"
	"time"

	"github.com/hayratyardim/donation_tracker/internal/models"
	"github.com/xuri/excelize/v2"
)

func ToExcel(donations []models.Donation) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Bağışlar"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, "", fmt.Errorf("sayfa oluşturma hatası: %w", err)
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := []string{
		"ID",
		"Tarih",
		"Saat",
		"Kanal",
		"Gönderen",
		"Kullanıcı Adı",
		"İçerik",
		"Mesaj Linki",
		"Takvime Eklendi",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#FFFFFF",
			Size:  11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
		},
	})

	linkStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color:     "#0563C1",
			Underline: "single",
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})

	calendarYesStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#006400",
			Bold:  true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
		},
	})

	calendarNoStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#8B0000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
		},
	})

	for i, d := range donations {
		row := i + 2

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), d.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), d.MessageDate.Format("02.01.2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), d.MessageDate.Format("15:04"))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), d.ChannelTitle)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), d.SenderName)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), d.SenderUser)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), d.Content)

		linkCell := fmt.Sprintf("H%d", row)
		f.SetCellValue(sheetName, linkCell, d.MessageLink)
		f.SetCellHyperLink(sheetName, linkCell, d.MessageLink, "External")
		f.SetCellStyle(sheetName, linkCell, linkCell, linkStyle)

		// Takvim durumu
		calendarCell := fmt.Sprintf("I%d", row)
		if d.AddedToCalendar {
			f.SetCellValue(sheetName, calendarCell, "✓ Evet")
			f.SetCellStyle(sheetName, calendarCell, calendarCell, calendarYesStyle)
		} else {
			f.SetCellValue(sheetName, calendarCell, "Hayır")
			f.SetCellStyle(sheetName, calendarCell, calendarCell, calendarNoStyle)
		}

		for j := 0; j < 7; j++ {
			cell := fmt.Sprintf("%c%d", 'A'+j, row)
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}
	}

	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 8)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 20)
	f.SetColWidth(sheetName, "F", "F", 15)
	f.SetColWidth(sheetName, "G", "G", 50)
	f.SetColWidth(sheetName, "H", "H", 40)
	f.SetColWidth(sheetName, "I", "I", 15)

	f.SetRowHeight(sheetName, 1, 25)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("excel yazma hatası: %w", err)
	}

	fileName := fmt.Sprintf("bagislar_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	return buffer.Bytes(), fileName, nil
}
