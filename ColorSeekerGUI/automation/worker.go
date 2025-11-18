package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
)

type Status struct {
	Timestamp time.Time
	Message   string
	Level     string
}

type Config struct {
	ColorX1        int     `json:"color_x1"`
	ColorY1        int     `json:"color_y1"`
	ColorX2        int     `json:"color_x2"`
	ColorY2        int     `json:"color_y2"`
	TargetColor    uint32  `json:"target_color"`
	ShadeVariation int     `json:"shade_variation"`
	GoodImagePath  string  `json:"good_image_path"`
	BadImagePath   string  `json:"bad_image_path"`
	LoopDelay      int     `json:"loop_delay_seconds"`
	MatchThreshold float64 `json:"match_threshold"`
	SearchScale    int     `json:"search_scale"`
	RefineRadius   int     `json:"refine_radius"`
}

func DefaultConfig() Config {
	return Config{
		ColorX1:        11,
		ColorY1:        420,
		ColorX2:        11,
		ColorY2:        440,
		TargetColor:    0x77604B,
		ShadeVariation: 10,
		GoodImagePath:  "Good.png",
		BadImagePath:   "bad.png",
		LoopDelay:      1,
		MatchThreshold: 0.80,
		SearchScale:    16,
		RefineRadius:   24,
	}
}

func LoadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return DefaultConfig(), err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), err
	}

	return config, nil
}

func (c *Config) Save(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func Run(ctx context.Context, config Config, statusChan chan<- Status) {
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			statusChan <- Status{
				Timestamp: time.Now(),
				Message:   "Автоматизация остановлена",
				Level:     "info",
			}
			return
		default:
			iteration++
			statusChan <- Status{
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("=== Итерация #%d ===", iteration),
				Level:     "info",
			}

			foundColor, foundY := findColorInArea(config, statusChan)

			if foundColor {
				statusChan <- Status{
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("✓ Цвет #%06X найден на Y=%d", config.TargetColor, foundY),
					Level:     "success",
				}
				statusChan <- Status{
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("→ Ищу изображение: %s", config.GoodImagePath),
					Level:     "info",
				}
				if findAndClickImage(config.GoodImagePath, config, statusChan) {
					statusChan <- Status{
						Timestamp: time.Now(),
						Message:   fmt.Sprintf("✓ Изображение %s найдено и кликнуто", config.GoodImagePath),
						Level:     "success",
					}
				} else {
					statusChan <- Status{
						Timestamp: time.Now(),
						Message:   fmt.Sprintf("✗ Изображение %s не найдено", config.GoodImagePath),
						Level:     "error",
					}
				}
			} else {
				statusChan <- Status{
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("✗ Цвет #%06X не найден", config.TargetColor),
					Level:     "warning",
				}
				statusChan <- Status{
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("→ Ищу изображение: %s", config.BadImagePath),
					Level:     "info",
				}
				if findAndClickImage(config.BadImagePath, config, statusChan) {
					statusChan <- Status{
						Timestamp: time.Now(),
						Message:   fmt.Sprintf("✓ Изображение %s найдено и кликнуто", config.BadImagePath),
						Level:     "success",
					}
				} else {
					statusChan <- Status{
						Timestamp: time.Now(),
						Message:   fmt.Sprintf("✗ Изображение %s не найдено", config.BadImagePath),
						Level:     "error",
					}
				}
			}

			time.Sleep(time.Duration(config.LoopDelay) * time.Second)
		}
	}
}

func findColorInArea(config Config, statusChan chan<- Status) (bool, int) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		statusChan <- Status{
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Ошибка захвата экрана: %v", err),
			Level:     "error",
		}
		return false, 0
	}

	targetR, targetG, targetB := hexToRGB(config.TargetColor)

	for y := config.ColorY1; y <= config.ColorY2; y++ {
		x := config.ColorX1
		if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}

		pixelColor := img.At(x, y)
		r, g, b, _ := pixelColor.RGBA()

		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)

		if colorMatch(r8, g8, b8, targetR, targetG, targetB, config.ShadeVariation) {
			return true, y
		}
	}

	return false, 0
}

func findAndClickImage(imagePath string, config Config, statusChan chan<- Status) bool {
	bounds := screenshot.GetDisplayBounds(0)
	screen, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Printf("Ошибка захвата экрана: %v\n", err)
		return false
	}

	template, err := loadImage(imagePath)
	if err != nil {
		log.Printf("Ошибка загрузки изображения %s: %v\n", imagePath, err)
		return false
	}

	loc, confidence := templateMatch(screen, template, config.SearchScale, config.RefineRadius)
	if confidence >= config.MatchThreshold {
		centerX := loc.X + template.Bounds().Dx()/2
		centerY := loc.Y + template.Bounds().Dy()/2

		robotgo.Move(centerX, centerY)
		time.Sleep(50 * time.Millisecond)
		robotgo.Click()

		statusChan <- Status{
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("  Клик: X=%d, Y=%d (точность: %.0f%%)", centerX, centerY, confidence*100),
			Level:     "info",
		}

		return true
	}

	return false
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func templateMatch(img image.Image, template image.Image, searchScale, refineRadius int) (image.Point, float64) {
	imgBounds := img.Bounds()
	tmplBounds := template.Bounds()

	bestLoc := image.Point{X: 0, Y: 0}
	bestScore := -1.0

	for y := imgBounds.Min.Y; y <= imgBounds.Max.Y-tmplBounds.Dy(); y += searchScale {
		for x := imgBounds.Min.X; x <= imgBounds.Max.X-tmplBounds.Dx(); x += searchScale {
			score := compareRegion(img, template, x, y)
			if score > bestScore {
				bestScore = score
				bestLoc = image.Point{X: x, Y: y}
			}
		}
	}

	refineLoc, refineScore := refineSearch(img, template, bestLoc, refineRadius)
	if refineScore > bestScore {
		bestScore = refineScore
		bestLoc = refineLoc
	}

	return bestLoc, bestScore
}

func refineSearch(img image.Image, template image.Image, center image.Point, radius int) (image.Point, float64) {
	imgBounds := img.Bounds()
	tmplBounds := template.Bounds()

	bestLoc := center
	bestScore := -1.0

	minY := max(imgBounds.Min.Y, center.Y-radius)
	maxY := min(imgBounds.Max.Y-tmplBounds.Dy(), center.Y+radius)
	minX := max(imgBounds.Min.X, center.X-radius)
	maxX := min(imgBounds.Max.X-tmplBounds.Dx(), center.X+radius)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			score := compareRegion(img, template, x, y)
			if score > bestScore {
				bestScore = score
				bestLoc = image.Point{X: x, Y: y}
			}
		}
	}

	return bestLoc, bestScore
}

func compareRegion(img image.Image, template image.Image, startX, startY int) float64 {
	tmplBounds := template.Bounds()
	width := tmplBounds.Dx()
	height := tmplBounds.Dy()

	var totalDiff float64
	var maxDiff float64 = float64(width * height * 255 * 3)

	for ty := 0; ty < height; ty++ {
		for tx := 0; tx < width; tx++ {
			imgColor := img.At(startX+tx, startY+ty)
			tmplColor := template.At(tmplBounds.Min.X+tx, tmplBounds.Min.Y+ty)

			ir, ig, ib, _ := imgColor.RGBA()
			tr, tg, tb, _ := tmplColor.RGBA()

			ir8 := uint8(ir >> 8)
			ig8 := uint8(ig >> 8)
			ib8 := uint8(ib >> 8)

			tr8 := uint8(tr >> 8)
			tg8 := uint8(tg >> 8)
			tb8 := uint8(tb >> 8)

			totalDiff += float64(abs(int(ir8) - int(tr8)))
			totalDiff += float64(abs(int(ig8) - int(tg8)))
			totalDiff += float64(abs(int(ib8) - int(tb8)))
		}
	}

	similarity := 1.0 - (totalDiff / maxDiff)
	return similarity
}

func hexToRGB(hex uint32) (uint8, uint8, uint8) {
	r := uint8((hex >> 16) & 0xFF)
	g := uint8((hex >> 8) & 0xFF)
	b := uint8(hex & 0xFF)
	return r, g, b
}

func colorMatch(r1, g1, b1, r2, g2, b2 uint8, tolerance int) bool {
	return abs(int(r1)-int(r2)) <= tolerance &&
		abs(int(g1)-int(g2)) <= tolerance &&
		abs(int(b1)-int(b2)) <= tolerance
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
