package main

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("Legal Hold Processor")
	
	// Input fields
	dataEntry := widget.NewEntry()
	dataEntry.SetPlaceHolder("Legal Hold Data ZIP File")
	
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Output Path")
	
	secretEntry := widget.NewEntry()
	secretEntry.SetPlaceHolder("Legal Hold Secret (Optional)")
	secretEntry.Password = true

	// File picker buttons
	selectDataBtn := widget.NewButton("Browse...", func() {
		fd := dialog.NewFileOpen(func(uri fyne.URIReadCloser, err error) {
			if uri == nil {
				return
			}
			dataEntry.SetText(uri.URI().Path())
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".zip"}))
		fd.Show()
	})

	selectOutputBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			outputEntry.SetText(uri.Path())
		}, w)
	})

	// Process button
	processBtn := widget.NewButton("Process Legal Hold", func() {
		if dataEntry.Text == "" {
			dialog.ShowError(errors.New("Legal hold data path is required"), w)
			return
		}
		if outputEntry.Text == "" {
			dialog.ShowError(errors.New("Output path is required"), w)
			return
		}
		
		// Call processing in goroutine to keep UI responsive
		go processLegalHold(dataEntry.Text, outputEntry.Text, secretEntry.Text)
	})

	// Layout
	dataContainer := container.NewBorder(nil, nil, nil, selectDataBtn, dataEntry)
	outputContainer := container.NewBorder(nil, nil, nil, selectOutputBtn, outputEntry)

	content := container.NewVBox(
		widget.NewLabel("Legal Hold Processor"),
		dataContainer,
		outputContainer,
		secretEntry,
		processBtn,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
