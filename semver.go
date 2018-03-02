package devtools

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/karlmutch/bump-ver/version"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

var (
	rVerReplace *regexp.Regexp
	rFind       *regexp.Regexp
	rHTML       *regexp.Regexp
)

func init() {
	r, errGo := regexp.Compile("\\<repo-version\\>.*?\\</repo-version\\>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rFind = r
	r, errGo = regexp.Compile("<[^>]*>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rHTML = r

	r, errGo = regexp.Compile("\\<repo-version\\>(.*?)\\</repo-version\\>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rVerReplace = r
}

func (md *MetaData) LoadVer(fn string) (ver *semver.Version, err errors.Error) {

	if md.SemVer != nil {
		return nil, errors.New("version already loaded").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	file, errGo := os.Open(fn)
	if errGo != nil {
		return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)

	for scan.Scan() {
		versions := rFind.FindAllString(scan.Text(), -1)
		if len(versions) == 0 {
			continue
		}
		for _, version := range versions {
			if ver == nil {
				ver, errGo = semver.NewVersion(html.UnescapeString(rHTML.ReplaceAllString(versions[0], "")))
				if errGo != nil {
					return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
				}
				continue
			}
			newVer := html.UnescapeString(rHTML.ReplaceAllString(version, ""))
			if newVer != ver.String() {
				return nil, errors.New("all repo-version HTML tags must have the same value").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
			}
		}
	}

	md.SemVer, errGo = semver.NewVersion(ver.String())
	if errGo != nil {
		md.SemVer = nil
		return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	return ver, nil
}

func (md *MetaData) Apply(files []string) (err errors.Error) {

	if len(files) == 0 {
		return errors.New("the apply command requires that files are specified with the -t option").With("stack", stack.Trace().TrimRuntime())
	}

	checkedFiles := make([]string, 0, len(files))
	for _, file := range files {
		if len(file) != 0 {
			if _, err := os.Stat(file); err != nil {
				fmt.Fprintf(os.Stderr, "a user specified target file was not found '%s'\n", file)
				continue
			}
			checkedFiles = append(checkedFiles, file)
		}
	}

	if len(checkedFiles) != len(files) {
		return errors.New("no usable targets were found to apply the version to").With("stack", stack.Trace().TrimRuntime())
	}

	// Process the files but stop on any errors
	for _, file := range checkedFiles {
		if err = md.Replace(file, file, false); err != nil {
			return err
		}
	}

	return nil
}

func (md *MetaData) Inject(file string) (err errors.Error) {

	if len(file) == 0 {
		return errors.New("the inject command requires that only a single target file is specified with the -t option").With("stack", stack.Trace().TrimRuntime())
	}

	if _, err := os.Stat(file); err != nil {
		return errors.New(fmt.Sprintf("a user specified target file was not found '%s'\n", file)).With("stack", stack.Trace().TrimRuntime())
	}

	// Process the file sto stdout but stop on any errors
	if err = md.Replace(file, "-", true); err != nil {
		return err
	}

	return nil
}

func (md *MetaData) Replace(fn string, dest string, substitute bool) (err errors.Error) {

	// To prevent destructive replacements first copy the file then modify the copy
	// and in an atomic operation copy the copy back over the original file, then
	// delete the working file
	origFn, errGo := filepath.Abs(fn)
	if errGo != nil {
		return errors.Wrap(errGo, "input file could not be resolved to an absolute file path").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	tmp, errGo := ioutil.TempFile(filepath.Dir(origFn), filepath.Base(origFn))
	if errGo != nil {
		return errors.Wrap(errGo, "temporary file could not be generated").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer func() {
		defer os.Remove(tmp.Name())

		tmp.Close()
	}()

	file, errGo := os.OpenFile(origFn, os.O_RDWR, 0600)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	newVer := fmt.Sprintf("<repo-version>%s</repo-version>", md.SemVer.String())
	if substitute {
		newVer = fmt.Sprintf("%s", md.SemVer.String())
	}

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		tmp.WriteString(rVerReplace.ReplaceAllString(scan.Text(), newVer) + "\n")
	}

	tmp.Sync()
	if fn == dest {
		defer file.Close()
	} else {
		file.Close()

		if dest == "-" {
			file = os.Stdout
		} else {
			file, errGo = os.OpenFile(dest, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
			if errGo != nil {
				return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
			}
			defer file.Close()
		}
	}

	if dest != "-" {
		if _, errGo = file.Seek(0, io.SeekStart); errGo != nil {
			return errors.Wrap(errGo, "failed to rewind the input file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
		}
	}
	if _, errGo = tmp.Seek(0, io.SeekStart); errGo != nil {
		return errors.Wrap(errGo, "failed to rewind a temporary file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	// Copy the output file on top of the original file
	written, errGo := io.Copy(file, tmp)
	if errGo != nil {
		return errors.Wrap(errGo, "failed to update the input file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	// Because we overwrote the file we need to trim off the end of the file if it shrank in size
	file.Truncate(written)

	return nil
}