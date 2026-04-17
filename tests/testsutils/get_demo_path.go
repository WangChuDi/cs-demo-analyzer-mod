package testsutils

import (
	"path/filepath"
	"runtime"
)

// GetDemoPath returns the path to the demo for testing.
func GetDemoPath(gameFolder string, name string) string {
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("..", "cs-demos", gameFolder, name+".dem")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFilePath), "..", "..", "cs-demos", gameFolder, name+".dem"))
}
