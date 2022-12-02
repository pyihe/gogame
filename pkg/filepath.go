package pkg

import "os"

// MakeDir 创建目录
func MakeDir(targetPath string) error {
	if _, err := os.Stat(targetPath); err != nil {
		if !os.IsExist(err) {
			//创建目录
			if mErr := os.MkdirAll(targetPath, os.ModePerm); mErr != nil {
				return mErr
			}
		}
	}
	return nil
}
