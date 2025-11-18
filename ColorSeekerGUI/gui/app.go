package gui

import (
        "context"
        "fmt"
        "image/color"
        "log"
        "strconv"
        "strings"
        "sync"
        "time"

        "gioui.org/app"
        "gioui.org/font"
        "gioui.org/font/gofont"
        "gioui.org/layout"
        "gioui.org/op"
        "gioui.org/text"
        "gioui.org/unit"
        "gioui.org/widget"
        "gioui.org/widget/material"

        "code-rewrite-runner/automation"
)

const configFile = "config.json"

type App struct {
        window     *app.Window
        theme      *material.Theme
        config     automation.Config
        statusChan chan automation.Status
        
        logMutex   sync.Mutex
        logBuffer  []string
        maxLogs    int

        startBtn widget.Clickable
        stopBtn  widget.Clickable

        colorX1Editor        widget.Editor
        colorX2Editor        widget.Editor
        colorY1Editor        widget.Editor
        colorY2Editor        widget.Editor
        targetColorEditor    widget.Editor
        loopDelayEditor      widget.Editor
        matchThresholdEditor widget.Editor

        running   bool
        cancelFn  context.CancelFunc
        statusCtx context.Context

        list widget.List
}

func NewApp() *App {
        config, err := automation.LoadConfig(configFile)
        if err != nil {
                log.Printf("Не удалось загрузить конфиг, использую значения по умолчанию: %v", err)
                config = automation.DefaultConfig()
        }

        th := material.NewTheme()
        th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

        a := &App{
                window:     app.NewWindow(app.Title("Code Rewrite Runner"), app.Size(unit.Dp(900), unit.Dp(700))),
                theme:      th,
                config:     config,
                statusChan: make(chan automation.Status, 100),
                logBuffer:  make([]string, 0, 200),
                maxLogs:    200,
                running:    false,
        }

        a.colorX1Editor.SingleLine = true
        a.colorX2Editor.SingleLine = true
        a.colorY1Editor.SingleLine = true
        a.colorY2Editor.SingleLine = true
        a.targetColorEditor.SingleLine = true
        a.loopDelayEditor.SingleLine = true
        a.matchThresholdEditor.SingleLine = true

        a.colorX1Editor.SetText(fmt.Sprintf("%d", config.ColorX1))
        a.colorX2Editor.SetText(fmt.Sprintf("%d", config.ColorX2))
        a.colorY1Editor.SetText(fmt.Sprintf("%d", config.ColorY1))
        a.colorY2Editor.SetText(fmt.Sprintf("%d", config.ColorY2))
        a.targetColorEditor.SetText(fmt.Sprintf("%06X", config.TargetColor))
        a.loopDelayEditor.SetText(fmt.Sprintf("%d", config.LoopDelay))
        a.matchThresholdEditor.SetText(fmt.Sprintf("%.0f", config.MatchThreshold*100))

        a.list.Axis = layout.Vertical

        return a
}

func (a *App) Run() error {
        var ops op.Ops
        
        go func() {
                for status := range a.statusChan {
                        logLine := fmt.Sprintf("[%s] %s",
                                status.Timestamp.Format("15:04:05"),
                                status.Message)
                        
                        a.logMutex.Lock()
                        a.logBuffer = append(a.logBuffer, logLine)
                        if len(a.logBuffer) > a.maxLogs {
                                a.logBuffer = a.logBuffer[1:]
                        }
                        a.logMutex.Unlock()
                        
                        a.window.Invalidate()
                }
        }()

        for {
                e := a.window.NextEvent()
                switch e := e.(type) {
                case app.DestroyEvent:
                        if a.cancelFn != nil {
                                a.cancelFn()
                        }
                        return e.Err

                case app.FrameEvent:
                        gtx := app.NewContext(&ops, e)
                        a.layout(gtx)
                        e.Frame(gtx.Ops)
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
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        return a.settingsPanel(gtx)
                }),
                layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
                        return a.logView(gtx)
                }),
        )
}

func (a *App) header(gtx layout.Context) layout.Dimensions {
        return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                return layout.Flex{
                        Axis:    layout.Vertical,
                        Spacing: layout.SpaceEnd,
                }.Layout(gtx,
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                title := material.H5(a.theme, "Code Rewrite Runner")
                                title.Alignment = text.Middle
                                return title.Layout(gtx)
                        }),
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                subtitle := material.Body2(a.theme, "Автоматизация поиска цвета и кликов по изображениям")
                                subtitle.Alignment = text.Middle
                                subtitle.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
                                return subtitle.Layout(gtx)
                        }),
                )
        })
}

func (a *App) controls(gtx layout.Context) layout.Dimensions {
        return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                return layout.Flex{
                        Axis:    layout.Horizontal,
                        Spacing: layout.SpaceEvenly,
                        Alignment: layout.Middle,
                }.Layout(gtx,
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                if a.startBtn.Clicked(gtx) && !a.running {
                                        a.applySettings()
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
                        layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                if a.stopBtn.Clicked(gtx) && a.running {
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
                        layout.Rigid(layout.Spacer{Width: unit.Dp(32)}.Layout),
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

func (a *App) settingsPanel(gtx layout.Context) layout.Dimensions {
        return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                return layout.Flex{
                        Axis: layout.Vertical,
                }.Layout(gtx,
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                label := material.Body1(a.theme, "Настройки:")
                                label.Font.Weight = font.Bold
                                return label.Layout(gtx)
                        }),
                        layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                return layout.Flex{
                                        Axis:    layout.Horizontal,
                                        Spacing: layout.SpaceBetween,
                                }.Layout(gtx,
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "X начало:", &a.colorX1Editor, 80)
                                        }),
                                        layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "X конец:", &a.colorX2Editor, 80)
                                        }),
                                        layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "Y начало:", &a.colorY1Editor, 80)
                                        }),
                                        layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "Y конец:", &a.colorY2Editor, 80)
                                        }),
                                )
                        }),
                        layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                return layout.Flex{
                                        Axis:    layout.Horizontal,
                                        Spacing: layout.SpaceBetween,
                                }.Layout(gtx,
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "Цвет (HEX):", &a.targetColorEditor, 150)
                                        }),
                                        layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "Интервал (сек):", &a.loopDelayEditor, 100)
                                        }),
                                        layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
                                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                                return a.inputField(gtx, "Порог совпад. (%):", &a.matchThresholdEditor, 100)
                                        }),
                                )
                        }),
                )
        })
}

func (a *App) inputField(gtx layout.Context, label string, editor *widget.Editor, width int) layout.Dimensions {
        return layout.Flex{
                Axis:      layout.Horizontal,
                Alignment: layout.Middle,
        }.Layout(gtx,
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        lbl := material.Body2(a.theme, label)
                        lbl.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
                        return lbl.Layout(gtx)
                }),
                layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        if a.running {
                                gtx = gtx.Disabled()
                        }
                        gtx.Constraints.Min.X = gtx.Dp(unit.Dp(width))
                        gtx.Constraints.Max.X = gtx.Dp(unit.Dp(width))
                        e := material.Editor(a.theme, editor, "")
                        e.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
                        return e.Layout(gtx)
                }),
        )
}

func (a *App) logView(gtx layout.Context) layout.Dimensions {
        return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                return layout.Flex{
                        Axis: layout.Vertical,
                }.Layout(gtx,
                        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                                label := material.Body2(a.theme, "Лог событий:")
                                label.Font.Weight = font.Bold
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
        a.logMutex.Lock()
        logText := ""
        start := 0
        if len(a.logBuffer) > 30 {
                start = len(a.logBuffer) - 30
        }
        for i := start; i < len(a.logBuffer); i++ {
                logText += a.logBuffer[i] + "\n"
        }
        a.logMutex.Unlock()

        if logText == "" {
                logText = "Нажмите START для запуска автоматизации...\n\nПараметры:\n- Область поиска цвета настраивается выше\n- Цвет в формате HEX (например, 77604B)\n- Интервал проверки в секундах\n- Порог совпадения от 0 до 100%"
        }

        label := material.Body2(a.theme, logText)
        label.Font.Typeface = "monospace"
        return label.Layout(gtx)
}

func (a *App) applySettings() {
        if colorX1, err := strconv.Atoi(a.colorX1Editor.Text()); err == nil {
                a.config.ColorX1 = colorX1
        }

        if colorX2, err := strconv.Atoi(a.colorX2Editor.Text()); err == nil {
                a.config.ColorX2 = colorX2
        }

        if colorY1, err := strconv.Atoi(a.colorY1Editor.Text()); err == nil {
                a.config.ColorY1 = colorY1
        }

        if colorY2, err := strconv.Atoi(a.colorY2Editor.Text()); err == nil {
                a.config.ColorY2 = colorY2
        }

        colorHex := strings.TrimPrefix(a.targetColorEditor.Text(), "0x")
        colorHex = strings.TrimPrefix(colorHex, "#")
        if color, err := strconv.ParseUint(colorHex, 16, 32); err == nil {
                a.config.TargetColor = uint32(color)
        }

        if loopDelay, err := strconv.Atoi(a.loopDelayEditor.Text()); err == nil && loopDelay > 0 {
                a.config.LoopDelay = loopDelay
        }

        if threshold, err := strconv.ParseFloat(a.matchThresholdEditor.Text(), 64); err == nil {
                a.config.MatchThreshold = threshold / 100.0
                if a.config.MatchThreshold > 1.0 {
                        a.config.MatchThreshold = 1.0
                }
                if a.config.MatchThreshold < 0.0 {
                        a.config.MatchThreshold = 0.0
                }
        }

        if err := a.config.Save(configFile); err != nil {
                log.Printf("Ошибка сохранения конфигурации: %v", err)
        }
}

func (a *App) startAutomation() {
        a.running = true
        ctx, cancel := context.WithCancel(context.Background())
        a.statusCtx = ctx
        a.cancelFn = cancel

        a.statusChan <- automation.Status{
                Timestamp: time.Now(),
                Message:   "=== ЗАПУСК АВТОМАТИЗАЦИИ ===",
                Level:     "info",
        }
        a.statusChan <- automation.Status{
                Timestamp: time.Now(),
                Message:   fmt.Sprintf("Область поиска: X=%d-%d, Y=%d-%d", a.config.ColorX1, a.config.ColorX2, a.config.ColorY1, a.config.ColorY2),
                Level:     "info",
        }
        a.statusChan <- automation.Status{
                Timestamp: time.Now(),
                Message:   fmt.Sprintf("Целевой цвет: #%06X (допуск: ±%d)", a.config.TargetColor, a.config.ShadeVariation),
                Level:     "info",
        }
        a.statusChan <- automation.Status{
                Timestamp: time.Now(),
                Message:   fmt.Sprintf("Изображения: Good=%s, Bad=%s", a.config.GoodImagePath, a.config.BadImagePath),
                Level:     "info",
        }
        a.statusChan <- automation.Status{
                Timestamp: time.Now(),
                Message:   fmt.Sprintf("Интервал проверки: %d сек, Порог совпад.: %.0f%%", a.config.LoopDelay, a.config.MatchThreshold*100),
                Level:     "info",
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
                Timestamp: time.Now(),
                Message:   "=== ОСТАНОВКА АВТОМАТИЗАЦИИ ===",
                Level:     "info",
        }
        log.Println("Автоматизация остановлена")
}
