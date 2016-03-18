package tarmaker

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Err struct {
	reason    string
	parentErr error
}

func NewErr(reason string, args ...interface{}) *Err {
	return &Err{reason: fmt.Sprintf(reason, args...)}
}

func (e *Err) SetParent(err error) *Err {
	e.parentErr = err
	return e
}

func (e Err) Parent() error {
	return e.parentErr
}

func (e Err) Error() string {
	if e.parentErr != nil {
		return fmt.Sprintf("%s, error: %s", e.reason, e.parentErr)
	}
	return e.reason
}

func Make(file, output, prefix string, artifacts []string) error {
	var (
		err error
		fd  = os.Stdout
	)

	if prefix != "" {
		if !strings.HasSuffix(prefix, "/") {
			return NewErr("prefix param should always contain leading slash, got: %s", prefix)
		}
		if strings.Contains(strings.TrimSuffix(prefix, "/"), "/") {
			return NewErr("prefix param cannot contain slashes except in the end: %s", prefix)
		}
	}

	if output != "-" {
		if fd, err = os.Create(output); err != nil {
			return err
		}
		defer fd.Close()
	}

	var fin io.Reader
	if file == "-" {
		fin = os.Stdin
	} else {
		fin, err = os.Open(file)
		if err != nil {
			return NewErr("Failed to open input file %s", file).SetParent(err)
		}
	}

	composeContent, err := ioutil.ReadAll(fin)
	if err != nil {
		return NewErr("Failed to read inpit file content %s", file).SetParent(err)
	}

	tw := tar.NewWriter(fd)

	// Add some files to the archive.
	type filesF struct {
		Name string
		Body []byte
	}

	var files = []filesF{
		{prefix + "compose.yml", composeContent},
	}

	for _, pat := range artifacts {
		matches := []string{pat}

		if containsWildcards(pat) {
			if matches, err = filepath.Glob(pat); err != nil {
				return NewErr("Failed to scan artifacts directory %s", pat).SetParent(err)
			}
		}

		for _, f := range matches {
			body, err := ioutil.ReadFile(f)
			if err != nil {
				return NewErr("Failed to read artifact file %s", f).SetParent(err)
			}

			files = append(files, filesF{
				Name: prefix + "artifacts/" + filepath.Base(f),
				Body: body,
			})
		}
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return NewErr("Failed write tar header on file %s", file.Name).SetParent(err)
		}
		if _, err := tw.Write(file.Body); err != nil {
			return NewErr("Failed write tar file body on file %s", file.Name).SetParent(err)
		}
	}

	return tw.Close()
}

func containsWildcards(name string) bool {
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '\\' {
			i++
		} else if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}
