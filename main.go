package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fatih/color"
)

const version = "0.13.1"

var ignoreList = []string{
	".git",
	".cache",
}

// exit code
const (
	ErrInitialize = iota + 1
	ErrMakeData
	ErrOutput
)

type option struct {
	root    string
	version bool
	verbose bool
	ignore  string
	nocolor bool
	dirs    bool
	full    bool
	abort   bool
	total   bool
}

var opt = &option{}

func init() {
	const lsep = string(filepath.ListSeparator)
	flag.StringVar(&opt.root, "root", "", "tree top")
	flag.BoolVar(&opt.version, "version", false, "print version")
	flag.BoolVar(&opt.verbose, "verbose", true, "with error log")
	flag.StringVar(&opt.ignore, "ignore", strings.Join(ignoreList, lsep), "ignore directory. list separator is '"+lsep+"'")
	flag.BoolVar(&opt.nocolor, "nocolor", false, "no color")
	flag.BoolVar(&opt.dirs, "dirs", false, "show directory only")
	flag.BoolVar(&opt.full, "full", false, "full path")
	flag.BoolVar(&opt.abort, "abort", false, "if find error then abort process")
	flag.BoolVar(&opt.total, "total", false, "prints total number of files and directories")
	flag.Parse()
	if flag.NArg() != 0 {
		if flag.NArg() == 1 && opt.root == "" {
			opt.root = flag.Arg(0)
		} else {
			fmt.Fprintln(os.Stderr, "invalid arguments:", flag.Args())
			os.Exit(ErrInitialize)
		}
	}
}

/// simple walk
/// TODO: consider
func walk(root string, logger *log.Logger) (tree map[string][]os.FileInfo, exitCode int) {
	tree = make(map[string][]os.FileInfo)
	wg := new(sync.WaitGroup)
	type response struct {
		dir   string
		infos []os.FileInfo
	}
	queue := make(chan string, 128)
	resch := make(chan *response, 128)
	errch := make(chan error, 128)

	// for clean up goroutine
	var (
		goCounter = 0
		done      = make(chan bool)
	)
	defer func() {
		for i := 0; i != goCounter; i++ {
			done <- true
		}
	}()

	// error handler
	goCounter++
	go func() {
		for {
			select {
			case err := <-errch:
				if os.IsPermission(err) || os.IsNotExist(err) {
					logger.Println(err)
					exitCode = ErrMakeData
				} else {
					panic(err)
				}
			case <-done:
				return
			}
		}
	}()

	// make map
	goCounter++
	go func() {
		for {
			select {
			case res := <-resch:
				if res != nil {
					if _, ok := tree[res.dir]; ok {
						logger.Println("duplicated directory:", res.dir)
					} else {
						tree[res.dir] = res.infos
					}
				}
				wg.Done()
			case <-done:
				return
			}
		}
	}()

	// worker
	maxWorker := runtime.NumCPU()
	if maxWorker < 1 {
		maxWorker = 1
	}
	goCounter += maxWorker
	for i := 0; i < maxWorker; i++ {
		go func() {
			for {
				select {
				case dir := <-queue:
					infos, err := ioutil.ReadDir(dir)
					if err != nil {
						errch <- err
						resch <- nil
						continue
					}
					resch <- &response{dir: dir, infos: infos}
					for _, info := range infos {
						if info.IsDir() {
							wg.Add(1)
							go func(dir string, info os.FileInfo) {
								queue <- filepath.Join(dir, info.Name())
							}(dir, info)
						}
					}
				case <-done:
					return
				}
			}
		}()
	}

	wg.Add(1)
	queue <- root
	wg.Wait()
	return tree, exitCode
}

func run(w, errw io.Writer, opt *option) int {
	if opt.version {
		fmt.Fprintf(w, "version %s\n", version)
		return 0
	}
	color.NoColor = opt.nocolor
	logger := log.New(errw, "[tree]:", log.Lshortfile)
	if !opt.verbose {
		logger.SetOutput(ioutil.Discard)
	}
	if full, err := filepath.Abs(opt.root); err != nil {
		fmt.Fprintln(errw, err)
		return ErrInitialize
	} else {
		opt.root = full
	}

	/// make data
	tree, exitCode := walk(opt.root, logger)
	if exitCode != 0 && opt.abort {
		return exitCode
	}

	/// show
	var (
		result     = make([]string, 0, len(tree)*2)
		pushResult func(string, int)
		depLine    = func(depth int) string {
			str := ""
			for i := 0; i != depth; i++ {
				str += " "
				if depth-i == 1 {
					str += fmt.Sprintf("- ")
					break
				}
			}
			return str
		}
		ignoreMap = func() map[string]bool {
			m := make(map[string]bool)
			for _, ipath := range filepath.SplitList(opt.ignore) {
				m[ipath] = true
			}
			return m
		}()
		nfiles uint
		ndirs  uint
	)
	var pathFilter func(string, os.FileInfo, int) string
	if opt.full {
		pathFilter = func(dir string, info os.FileInfo, depth int) string { return filepath.Join(dir, info.Name()) }
	} else {
		pathFilter = func(dir string, info os.FileInfo, depth int) string { return depLine(depth) + info.Name() }
	}
	pushResult = func(dir string, depth int) {
		var info os.FileInfo
		for _, info = range tree[dir] {
			path := pathFilter(dir, info, depth)
			if info.IsDir() {
				if ignoreMap[info.Name()] {
					result = append(result, color.RedString("%s%c", path, filepath.Separator))
				} else {
					result = append(result, color.CyanString("%s%c", path, filepath.Separator))
					pushResult(filepath.Join(dir, info.Name()), depth+1)
				}
				ndirs++
				continue
			}
			if opt.dirs {
				continue
			}
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				result = append(result, color.GreenString("%s", path))
			} else {
				result = append(result, path)
			}
			nfiles++
		}
	}
	pushResult(opt.root, 0)
	_, err := fmt.Fprintln(w, strings.Join(result, "\n"))
	if opt.total {
		_, err = fmt.Fprintf(w, "\ndirectory %d\nfile %d\n", ndirs, nfiles)
	}
	if err != nil {
		fmt.Fprintln(errw, err)
		return ErrOutput
	}
	return exitCode
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, opt))
}
