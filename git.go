package devtools

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack

	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// This file contains some utility functions for extracting and using git information

func (md *MetaData) LoadGit(dir string, scanParents bool) (err errors.Error) {
	if md.Git != nil {
		return errors.New("git info already loaded, set Git member to nil if new information desired").With("stack", stack.Trace().TrimRuntime())
	}

	gitDir := dir
	for {
		_, errGo := os.Stat(filepath.Join(gitDir, ".git"))
		if errGo == nil {
			break
		}
		if !scanParents {
			return errors.Wrap(errGo, "does not appear to be the top directory of a git repo").With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
		}
		gitDir = filepath.Dir(gitDir)
		if len(gitDir) == 0 {
			return errors.Wrap(errGo, "could not locate a git repo in the directory heirarchy").With("stack", stack.Trace().TrimRuntime()).With("dir", dir)
		}
	}

	md.Git = &GitInfo{
		Dir: gitDir,
	}

	repo, errGo := git.PlainOpen(gitDir)
	if errGo != nil {
		md.Git.Err = errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
		return md.Git.Err
	}
	ref, errGo := repo.Head()
	if errGo != nil {
		md.Git.Err = errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
		return md.Git.Err
	}

	splits := strings.Split(ref.Name().String(), "/")

	md.Git.Branch = splits[len(splits)-1]
	md.Git.Repo = repo
	refs, _ := repo.Remotes()
	md.Git.URL = refs[0].Config().URLs[0]

	// Now try to find the first tag that matches the current HEAD
	head, _ := md.Git.Repo.Head()

	tags, _ := md.Git.Repo.Tags()
	_ = tags.ForEach(func(t *plumbing.Reference) error {
		if head.Hash() == t.Hash() {
			splits := strings.Split(t.Name().String(), "/")
			md.Git.Tag = splits[len(splits)-1]
		}
		return nil
	})

	return nil
}
