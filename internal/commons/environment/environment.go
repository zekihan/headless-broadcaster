package environment

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvironmentFile(others ...string) error {
	envFile := ".env"

	err := loadFile(envFile)
	if err != nil {
		return err
	}

	overrideEnvFile := ".override.env"

	err = loadFile(overrideEnvFile)
	if err != nil {
		return err
	}

	for _, filePath := range others {
		if filePath == "" {
			continue
		}

		err = loadFile(filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("error checking %s file: %w", filePath, err)
	}

	if info.IsDir() {
		return nil
	}

	err = godotenv.Overload(filePath)
	if err != nil {
		return fmt.Errorf("error loading %s file: %w", filePath, err)
	}

	return nil
}
