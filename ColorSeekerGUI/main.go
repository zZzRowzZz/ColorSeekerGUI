package main

import (
	"log"
	"os"

	"code-rewrite-runner/gui"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	if err := checkImageFiles(); err != nil {
		log.Printf("Внимание: %v", err)
		log.Println("Программа продолжит работу, но убедитесь что файлы Good.png и bad.png существуют перед запуском автоматизации.")
	}

	app := gui.NewApp()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func checkImageFiles() error {
	files := []string{"Good.png", "bad.png"}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
