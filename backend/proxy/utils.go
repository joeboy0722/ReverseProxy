package proxy

import (
	"path/filepath"
	"os"
)

// getExeDir 取得當前執行檔所在的絕對路徑資料夾
func getExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	evalPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return filepath.Dir(exePath)
	}
	return filepath.Dir(evalPath)
}
