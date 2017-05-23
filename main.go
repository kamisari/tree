package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const version = "0.2"

type option struct {
	root    string
	version bool
	ignore  string
}

var opt option

func init() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("tree:")
	log.SetFlags(log.Lshortfile)

	flag.StringVar(&opt.root, "root", "", "")
	flag.BoolVar(&opt.version, "version", false, "")
	flag.StringVar(&opt.ignore, "ignore", ".git", "list separatoer is '"+string(filepath.ListSeparator)+"'")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("invalid flag:", flag.Args())
	}
	if opt.version {
		fmt.Printf("version %s\n", version)
		os.Exit(0)
	}

	var err error
	if opt.root == "" {
		opt.root, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		opt.root, err = filepath.Abs(opt.root)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func run(root string, ignore string) {
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

	checkDir := func(dir string, ignoreList []string) bool {
		for _, t := range ignoreList {
			if dir == t {
				return false
			}
		}
		return true
	}

	checkDuple := make(map[string]bool)
	tree := ""
	deps := 0
	var push func(string)

	push = func(dir string) {
		defer func() {
			deps--
		}()
		if _, ok := checkDuple[dir]; ok {
			log.Println("duple check:", dir)
			return
		}
		checkDuple[dir] = true
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Println(err)
			return
		}
		for _, info := range infos {
			if info.IsDir() && checkDir(info.Name(), filepath.SplitList(ignore)) {
				tree += fmt.Sprintf("%s%s%c\n", depLine(deps), info.Name(), filepath.Separator)
				deps++
				push(filepath.Join(dir, info.Name()))
				continue
			}
			tree += fmt.Sprintf("%s%s\n", depLine(deps), info.Name())
		}
	}

	push(root)

	fmt.Println(tree)
}

func main() {
	run(opt.root, opt.ignore)
}
