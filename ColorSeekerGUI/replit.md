# Code Rewrite Runner - GUI Version с настраиваемыми параметрами

### Overview
This project is a desktop application built with Go and the Gio GUI framework. Its primary purpose is to automate mouse clicks based on color and image detection on the screen. The application features a fully customizable graphical user interface where all parameters can be configured, and the configuration is automatically saved. It targets Windows 11 as the primary platform and aims to provide a simple, portable, and efficient automation tool without complex external dependencies.

### User Preferences
- **Язык**: Русский
- **ОС**: Windows 11
- **Задача**: Автоматизация кликов на основе анализа цвета и изображений
- **Требования**:
  - Простая компиляция без сложных зависимостей
  - Настройка всех параметров через GUI
  - Сохранение настроек между запусками
  - Детальная информация о работе в логе

### System Architecture
The application is structured into `automation` and `gui` packages, separating the core logic from the user interface.
The core automation loop involves:
1. Loading configuration from `config.json` (or using defaults).
2. Displaying a GUI with editable fields for user input.
3. Capturing a specified rectangular screen area.
4. Searching for a target color within the captured area with a defined shade variation.
5. If the color is found, it searches for and clicks on `Good.png`.
6. If the color is not found, it searches for and clicks on `bad.png`.
7. Detailed logging of all actions, including timestamps, found objects, click coordinates, and match accuracy, with the last 30 lines displayed in the GUI.

**Key Features & Design Decisions:**
- **GUI-driven Configuration**: All parameters (search area coordinates, target HEX color, check frequency, image matching threshold) are editable directly within the GUI.
- **Automatic Configuration Saving**: Settings are persistently stored in `config.json`.
- **Pure Go Implementation**: Dependency on OpenCV has been removed, utilizing a custom Pure Go template matching algorithm for image detection.
- **Robust Logging**: Comprehensive logging provides detailed operational insights within the GUI.
- **Portability**: Compiles into a single executable file, simplifying distribution and execution.
- **Graceful Error Handling**: The application handles missing configuration files, images, and screen capture errors gracefully, ensuring continued operation.
- **UI/UX**: Features visual status indicators (e.g., button colors), disables fields during operation, provides informative hints, and includes automatic log scrolling.

**Project Structure:**
```
code-rewrite-runner/
├── automation/           # Automation logic (color/image search, clicks, config)
│   └── worker.go
├── gui/                 # Graphical User Interface
│   └── app.go
├── main.go             # Entry point
├── go.mod              # Module dependencies
├── build.bat           # Windows compilation script
├── Good.png            # User-replaceable image template
├── bad.png             # User-replaceable image template
├── config.json         # Auto-generated configuration file
└── README.md           # Project documentation
```

### External Dependencies
- `gioui.org v0.4.1`: Pure Go GUI framework (код обновлен под новый API).
- `github.com/kbinani/screenshot`: For screen capturing.
- `github.com/go-vgo/robotgo`: For mouse control and clicks.
- `encoding/json`: Standard library for JSON configuration saving/loading.
- **Pure Go template matching**: Custom, in-house implementation for image recognition, replacing OpenCV.

### Recent Updates
- ✅ Исправлены все ошибки компиляции с Gio v0.4.1 API:
  - `material.NewTheme()` теперь использует отдельный Shaper с gofont.Collection()
  - `window.Events()` заменен на `window.NextEvent()` (блокирующий вызов)
  - Event loop переписан: statusChan обрабатывается в отдельной goroutine
  - `Clicked()` теперь требует параметр `gtx`
  - `text.Bold` заменен на `font.Bold` из пакета `gioui.org/font`
  - Удален неиспользуемый импорт `gioui.org/io/system`