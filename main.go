package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
)

const version = "0.11"

type option struct {
	root    string
	version bool
	ignore  string
	noColor bool
	dirs    bool
	full    bool
	nolog   bool
}

var opt option

func init() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("tree:")
	log.SetFlags(log.Lshortfile)

	const lsep = string(filepath.ListSeparator)
	flag.StringVar(&opt.root, "root", "", "tree top")
	flag.BoolVar(&opt.version, "version", false, "")
	flag.StringVar(&opt.ignore, "ignore", ".git"+lsep+".cache", "ignore directory, list separator is '"+lsep+"'")
	flag.BoolVar(&opt.noColor, "nocolor", false, "")
	flag.BoolVar(&opt.dirs, "dirs", false, "show directory only")
	flag.BoolVar(&opt.full, "full", false, "full path")
	flag.BoolVar(&opt.nolog, "nolog", false, "no error log output")
	flag.Parse()
	if opt.version {
		fmt.Printf("version %s\n", version)
		os.Exit(0)
	}
	if flag.NArg() != 0 {
		if flag.NArg() == 1 {
			opt.root = flag.Arg(0)
		} else {
			log.Fatal("invalid argument:", flag.Args())
		}
	}
	color.NoColor = opt.noColor
	var err error
	opt.root, err = filepath.Abs(opt.root)
	if err != nil {
		log.Fatal(err)
	}
	if opt.nolog {
		null, err := os.Open(os.DevNull)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(null)
	}
}

func run(root string, ignore string, dirsonly bool, fullpath bool) int {
	wg := new(sync.WaitGroup)
	mux := new(sync.Mutex)
	exitCode := 0
	tree := make(map[string][]os.FileInfo)

	var pushTree func(string)
	pushTree = func(dir string) {
		defer wg.Done()

		mux.Lock()
		if _, ok := tree[dir]; ok {
			mux.Unlock()
			log.Println("ignore duplicate check:", dir)
			return
		}
		infos, err := ioutil.ReadDir(dir) // need mutex for countermove `too many open files`
		if err != nil {
			exitCode = 3
			mux.Unlock()
			log.Println(err)
			return
		}
		tree[dir] = infos
		mux.Unlock()

		for _, info := range infos {
			if info.IsDir() {
				wg.Add(1)
				go pushTree(filepath.Join(dir, info.Name()))
			}
		}
	}
	wg.Add(1)
	go pushTree(root)
	wg.Wait()

	/// show
	depLine := func(depth int) string {
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
	isIgnore := func(dir string, ignoreList []string) bool {
		for _, t := range ignoreList {
			if dir == t {
				return true
			}
		}
		return false
	}
	var result []string
	var pushResult func(string, int)
	pushResult = func(dir string, depth int) {
		for _, info := range tree[dir] {
			var path string
			if fullpath {
				path = filepath.Join(dir, info.Name())
			} else {
				path = depLine(depth) + info.Name()
			}
			if info.IsDir() {
				if isIgnore(info.Name(), filepath.SplitList(ignore)) {
					result = append(result, color.RedString("%s%c", path, filepath.Separator))
				} else {
					result = append(result, color.CyanString("%s%c", path, filepath.Separator))
					pushResult(filepath.Join(dir, info.Name()), depth+1)
				}
				continue
			}
			if dirsonly {
				continue
			}
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				result = append(result, color.GreenString("%s", path))
			} else {
				result = append(result, fmt.Sprintf("%s", path))
			}
		}
	}

	pushResult(root, 0)
	fmt.Println(strings.Join(result, "\n"))
	return exitCode
}

func main() {
	os.Exit(run(opt.root, opt.ignore, opt.dirs, opt.full))
}
