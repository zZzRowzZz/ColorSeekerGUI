package gui

import (
	"context"
	"fmt"
	"image/color"
	"log"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"code-rewrite-runner/automation"
)

type App struct {
	window     *app.Window
	theme      *material.Theme
	config     automation.Config
	statusChan chan automation.Status
	logBuffer  []string
	maxLogs    int

	startBtn widget.Clickable
	stopBtn  widget.Clickable

	running   bool
	cancelFn  context.CancelFunc
	statusCtx context.Context
}

func NewApp() *App {
	return &App{
		window:     app.NewWindow(app.Title("Code Rewrite Runner"), app.Size(unit.Dp(800), unit.Dp(600))),
		theme:      material.NewTheme(gofont.Collection()),
		config:     automation.DefaultConfig(),
		statusChan: make(chan automation.Status, 100),
		logBuffer:  make([]string, 0, 200),
		maxLogs:    200,
		running:    false,
	}
}

func (a *App) Run() error {
	var ops op.Ops
	for {
		select {
		case status := <-a.statusChan:
			logLine := fmt.Sprintf("[%s] %s",
				status.Timestamp.Format("15:04:05"),
				status.Message)
			a.logBuffer = append(a.logBuffer, logLine)
			if len(a.logBuffer) > a.maxLogs {
				a.logBuffer = a.logBuffer[1:]
			}
			a.window.Invalidate()

		case e := <-a.window.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				if a.cancelFn != nil {
					a.cancelFn()
				}
				return e.Err

			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				a.layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}
}

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.header(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.controls(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.logView(gtx)
		}),
	)
}

func (a *App) header(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		title := material.H5(a.theme, "Code Rewrite Runner")
		title.Alignment = text.Middle
		return title.Layout(gtx)
	})
}

func (a *App) controls(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Horizontal,
			Spacing: layout.SpaceEvenly,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if a.startBtn.Clicked() && !a.running {
					a.startAutomation()
				}
				btn := material.Button(a.theme, &a.startBtn, "START")
				btn.Background = color.NRGBA{R: 0, G: 150, B: 0, A: 255}
				if a.running {
					btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
					gtx = gtx.Disabled()
				}
				return btn.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if a.stopBtn.Clicked() && a.running {
					a.stopAutomation()
				}
				btn := material.Button(a.theme, &a.stopBtn, "STOP")
				btn.Background = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
				if !a.running {
					btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
					gtx = gtx.Disabled()
				}
				return btn.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				status := "Остановлен"
				statusColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255}
				if a.running {
					status = "Работает"
					statusColor = color.NRGBA{R: 0, G: 150, B: 0, A: 255}
				}
				label := material.H6(a.theme, fmt.Sprintf("Статус: %s", status))
				label.Color = statusColor
				return label.Layout(gtx)
			}),
		)
	})
}

func (a *App) logView(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(a.theme, "Лог событий:")
				label.Font.Weight = text.Bold
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.renderLogs(gtx)
			}),
		)
	})
}

func (a *App) renderLogs(gtx layout.Context) layout.Dimensions {
	logText := ""
	start := 0
	if len(a.logBuffer) > 30 {
		start = len(a.logBuffer) - 30
	}
	for i := start; i < len(a.logBuffer); i++ {
		logText += a.logBuffer[i] + "\n"
	}

	label := material.Body2(a.theme, logText)
	label.Font.Typeface = "monospace"
	return label.Layout(gtx)
}

func (a *App) startAutomation() {
	a.running = true
	ctx, cancel := context.WithCancel(context.Background())
	a.statusCtx = ctx
	a.cancelFn = cancel

	a.statusChan <- automation.Status{
		Message: "=== ЗАПУСК АВТОМАТИЗАЦИИ ===",
		Level:   "info",
	}
	a.statusChan <- automation.Status{
		Message: fmt.Sprintf("Область поиска: X=%d, Y=%d-%d", a.config.ColorX1, a.config.ColorY1, a.config.ColorY2),
		Level:   "info",
	}
	a.statusChan <- automation.Status{
		Message: fmt.Sprintf("Целевой цвет: #%06X", a.config.TargetColor),
		Level:   "info",
	}
	a.statusChan <- automation.Status{
		Message: fmt.Sprintf("Изображения: Good=%s, Bad=%s", a.config.GoodImagePath, a.config.BadImagePath),
		Level:   "info",
	}

	go automation.Run(ctx, a.config, a.statusChan)

	log.Println("Автоматизация запущена")
}

func (a *App) stopAutomation() {
	if a.cancelFn != nil {
		a.cancelFn()
		a.cancelFn = nil
	}
	a.running = false
	a.statusChan <- automation.Status{
		Message: "=== ОСТАНОВКА АВТОМАТИЗАЦИИ ===",
		Level:   "info",
	}
	log.Println("Автоматизация остановлена")
}
