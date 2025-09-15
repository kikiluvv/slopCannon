package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/kikiluvv/slopCannon/internal/video"
)

type Clip struct {
	Start float64
	End   float64
}

var clips []Clip

func RunGUI() {
	myApp := app.NewWithID("slopCannon")
	w := myApp.NewWindow("💣 slopCannon Editor 💣")
	w.Resize(fyne.NewSize(600, 400))

	var videoPath string
	var currentStart float64
	var currentEnd float64

	videoLabel := widget.NewLabel("No video loaded")
	timestampLabel := widget.NewLabel("Current: 0.0s")
	slider := widget.NewSlider(0, 100) // placeholder, updated when video loads

	slider.OnChanged = func(val float64) {
		timestampLabel.SetText(fmt.Sprintf("Current: %.2fs", val))
	}

	startButton := widget.NewButton("Mark Start", func() {
		currentStart = slider.Value
		fmt.Println("Start marked at", currentStart)
	})

	endButton := widget.NewButton("Mark End", func() {
		currentEnd = slider.Value
		fmt.Println("End marked at", currentEnd)
	})

	addClipButton := widget.NewButton("Add Clip", func() {
		if currentEnd > currentStart {
			clips = append(clips, Clip{Start: currentStart, End: currentEnd})
			fmt.Printf("Clip added: %.2f -> %.2f\n", currentStart, currentEnd)
		} else {
			fmt.Println("Invalid clip: end must be after start")
		}
	})

	printClipsButton := widget.NewButton("Print Clips", func() {
		fmt.Println("Current clips:", clips)
	})

	loadButton := widget.NewButton("Load Video", func() {
		fd := dialog.NewFileOpen(
			func(ur fyne.URIReadCloser, err error) {
				if ur == nil {
					return
				}
				videoPath = ur.URI().Path()
				fmt.Println("Video loaded:", videoPath)
				videoLabel.SetText("Loaded: " + videoPath)

				// get actual video duration
				duration, err := video.GetVideoDuration(videoPath)
				if err != nil {
					fmt.Println("Error getting video duration:", err)
					return
				}
				slider.Min = 0
				slider.Max = duration
				slider.Value = 0
				fmt.Println("Video duration:", duration, "seconds")
			}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".mov", ".mkv"}))
		fd.Show()
	})

	renderButton := widget.NewButton("Render Clips", func() {
		for i, c := range clips {
			err := video.RenderClip(videoPath, c.Start, c.End, "assets/mc.mp4", i+1)
			if err != nil {
				fmt.Println("Error rendering clip:", err)
			} else {
				fmt.Printf("Clip %d rendered successfully\n", i+1)
			}
		}
	})

	w.SetContent(
		container.NewVBox(
			videoLabel,
			slider,
			timestampLabel,
			container.NewHBox(startButton, endButton, addClipButton, printClipsButton),
			loadButton,
			renderButton,
		),
	)

	w.ShowAndRun()
}
