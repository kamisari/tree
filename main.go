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

const version = "0.9"

type option struct {
	root    string
	version bool
	ignore  string
	noColor bool
}

var opt option

func init() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("tree:")
	log.SetFlags(log.Lshortfile)

	const lsep = string(filepath.ListSeparator)
	flag.StringVar(&opt.root, "root", "", "tree top")
	flag.BoolVar(&opt.version, "version", false, "")
	flag.StringVar(&opt.ignore, "ignore", ".git"+lsep+".cache", "ignore directory, list separatoer is '"+lsep+"'")
	flag.BoolVar(&opt.noColor, "nocolor", false, "")
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
}

func run(root string, ignore string) (exitCode int) {
	wg := new(sync.WaitGroup)
	mux := new(sync.Mutex)
	checkDuple := make(map[string]bool)
	tree := make(map[string][]os.FileInfo)

	var pushTree func(string)
	pushTree = func(dir string) {
		defer wg.Done()

		mux.Lock()
		if checkDuple[dir] {
			mux.Unlock()
			log.Println("ignore duplicate check:", dir)
			return
		}
		checkDuple[dir] = true
		infos, err := ioutil.ReadDir(dir) // need mutex for countermove `too many open files`
		mux.Unlock()
		if err != nil {
			log.Println(err)
			exitCode = 3
			return
		}

		mux.Lock()
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
	depLine := func(deps int) string {
		str := ""
		for i := 0; i != deps; i++ {
			str += " "
			if deps-i == 1 {
				str += fmt.Sprintf("- ")
				break
			}
		}
		return str
	}
	isNotIgnore := func(dir string, ignoreList []string) bool {
		for _, t := range ignoreList {
			if dir == t {
				return false
			}
		}
		return true
	}
	var deps int
	var result []string
	var pushResult func(string)
	pushResult = func(dir string) {
		defer func() { deps-- }()
		for _, info := range tree[dir] {
			if info.IsDir() {
				if isNotIgnore(info.Name(), filepath.SplitList(ignore)) {
					result = append(result, color.CyanString("%s%s%c", depLine(deps), info.Name(), filepath.Separator))
					deps++
					pushResult(filepath.Join(dir, info.Name()))
					continue
				}
				result = append(result, color.RedString("%s%s%c", depLine(deps), info.Name(), filepath.Separator))
				continue
			}
			result = append(result, fmt.Sprintf("%s%s", depLine(deps), info.Name()))
		}
	}

	pushResult(root)
	fmt.Println(strings.Join(result, "\n"))
	return
}

func main() {
	os.Exit(run(opt.root, opt.ignore))
}
