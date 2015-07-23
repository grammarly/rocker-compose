package test

import (
	"io/ioutil"
	"os"
	"path"
)

func MakeFiles(baseDir string, files map[string]string) (err error) {
	for name, content := range files {
		fullName := path.Join(baseDir, name)
		dirName := path.Dir(fullName)
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			return
		}
		err = ioutil.WriteFile(fullName, []byte(content), 0644)
		if err != nil {
			return
		}
	}
	return
}
