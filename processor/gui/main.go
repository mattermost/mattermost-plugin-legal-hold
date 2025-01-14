package main

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"os/exec"
	"runtime"
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

	// Output text area
	outputText := widget.NewTextGrid()
	outputText.SetText("Processing output will appear here...")

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
		
		// Clear previous output
		outputText.SetText("")
		
		// Disable button while processing
		processBtn.Disable()
		processBtn.SetText("Processing...")
		
		// Call processing in goroutine to keep UI responsive
		go func() {
			indexPath, err := processLegalHold(dataEntry.Text, outputEntry.Text, secretEntry.Text, func(text string) {
				// Update UI in main thread
				current := outputText.Text()
				outputText.SetText(current + text)
				outputText.Refresh()
			})
			
			// Re-enable button when done
			processBtn.Enable()
			processBtn.SetText("Process Legal Hold")
			
			if err != nil {
				dialog.ShowError(err, w)
				openOutputBtn.Hide()
			} else {
				// Show and configure open output button
				openOutputBtn.OnTapped = func() {
					var cmd *exec.Cmd
					switch runtime.GOOS {
					case "darwin":
						cmd = exec.Command("open", indexPath)
					case "windows":
						cmd = exec.Command("cmd", "/c", "start", indexPath)
					default: // linux/unix
						cmd = exec.Command("xdg-open", indexPath)
					}
					if err := cmd.Run(); err != nil {
						dialog.ShowError(err, w)
					}
				}
				openOutputBtn.Show()
			}
		}()
	})

	// Layout
	dataContainer := container.NewBorder(nil, nil, nil, selectDataBtn, dataEntry)
	outputContainer := container.NewBorder(nil, nil, nil, selectOutputBtn, outputEntry)

	topContent := container.NewVBox(
		widget.NewLabel("Legal Hold Processor"),
		dataContainer,
		outputContainer,
		secretEntry,
		processBtn,
		widget.NewLabel("Processing Output:"),
	)

	// Create (initially hidden) open output button
	openOutputBtn := widget.NewButton("Open Output", nil)
	openOutputBtn.Hide()

	content := container.NewBorder(
		topContent, openOutputBtn, nil, nil,
		container.NewScroll(outputText),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
