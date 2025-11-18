@echo off
chcp 65001 >nul
echo ========================================
echo  Компиляция Code Rewrite Runner
echo  (GUI версия с кнопками START/STOP)
echo ========================================
echo.

echo [1/3] Проверка Go...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go не найден! Установите Go с https://golang.org/dl/
    pause
    exit /b 1
)
echo ✅ Go установлен
go version
echo.

echo [2/3] Загрузка зависимостей...
go mod download
if %errorlevel% neq 0 (
    echo ❌ Ошибка загрузки зависимостей!
    pause
    exit /b 1
)
echo ✅ Зависимости загружены
echo.

echo [3/3] Компиляция GUI приложения...
go build -ldflags "-s -w -H=windowsgui" -o CodeRewriteRunner.exe .
if %errorlevel% neq 0 (
    echo.
    echo ❌ Ошибка компиляции!
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo ✅ Успешно! Файл: CodeRewriteRunner.exe
echo ========================================
echo.
echo ⚠️ ВАЖНО: Перед запуском замените файлы:
echo - Good.png - точный скриншот вашего элемента
echo - bad.png - точный скриншот вашего элемента
echo.
echo Для запуска дважды кликните: CodeRewriteRunner.exe
echo Программа откроет графическое окно с кнопками
echo.
pause
