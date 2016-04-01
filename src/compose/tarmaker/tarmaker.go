package tarmaker

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/grammarly/rocker/src/template"
)

// Err is an error type for tarmaker package that wraps parent errors
type Err struct {
	reason    string
	parentErr error
}

type MakeTarOptions struct {
	File   string
	Output string
	Prefix string
	Vars   template.Vars
}

// NewErr makes Err
func NewErr(reason string, args ...interface{}) *Err {
	return &Err{reason: fmt.Sprintf(reason, args...)}
}

// SetParent sets parent error for Err
func (e *Err) SetParent(err error) *Err {
	e.parentErr = err
	return e
}

// Parent returns parent error of Err
func (e Err) Parent() error {
	return e.parentErr
}

// Error implements error interface
func (e Err) Error() string {
	if e.parentErr != nil {
		return fmt.Sprintf("%s, error: %s", e.reason, e.parentErr)
	}
	return e.reason
}

// Make makes a tar out of compose.yml file and a set of artifacts
func MakeTar(tm MakeTarOptions) error {
	var (
		err error
		fd  = os.Stdout
	)

	if tm.Prefix != "" {
		if !strings.HasSuffix(tm.Prefix, "/") {
			return NewErr("prefix param should always contain leading slash, got: %s", tm.Prefix)
		}
		if strings.Contains(strings.TrimSuffix(tm.Prefix, "/"), "/") {
			return NewErr("prefix param cannot contain slashes except in the end: %s", tm.Prefix)
		}
	}

	if tm.Output != "-" {
		if fd, err = os.Create(tm.Output); err != nil {
			return err
		}
		defer fd.Close()
	}

	var fin io.Reader
	if tm.File == "-" {
		fin = os.Stdin
	} else {
		fin, err = os.Open(tm.File)
		if err != nil {
			return NewErr("Failed to open input file %s", tm.File).SetParent(err)
		}
	}

	composeContent, err := ioutil.ReadAll(fin)
	if err != nil {
		return NewErr("Failed to read inpit file content %s", tm.File).SetParent(err)
	}

	tw := tar.NewWriter(fd)

	// Add some files to the archive.
	type filesF struct {
		Name string
		Body []byte
	}

	var files = []filesF{
		{tm.Prefix + "compose.yml", composeContent},
	}

	//Add variables to tarball.
	if len(tm.Vars) != 0 {
		varsBody, err := yaml.Marshal(tm.Vars)
		if err != nil {
			return NewErr("Failed to encode incoming variables to yaml").SetParent(err)
		}
		files = append(files, filesF{
			Name: tm.Prefix + "variables.yml",
			Body: varsBody,
		})
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
