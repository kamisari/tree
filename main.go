package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type dirInfo struct {
	infos []os.FileInfo
	deps  int
}
type treeMAP map[string]dirInfo

func try1(root string) {
	tree := make(treeMAP)
	//wg := new(sync.WaitGroup)
	//mux := new(sync.Mutex)
	var push func(string, int)
	push = func(dir string, deps int) {
		//defer wg.Done()
		if _, ok := tree[dir]; ok {
			return
		}
		if deps < 0 {
			panic("invalid deps: deps < 0")
		}
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			return
		}
		tree[dir] = dirInfo{infos: infos, deps: deps}
		for _, info := range infos {
			if info.IsDir() {
				//wg.Add(1)
				//go push(filepath.Join(dir, info.Name()))
				push(filepath.Join(dir, info.Name()), deps+1)
			}
		}
	}

	//wg.Add(1)
	push(root, 0)
	//wg.Wait()

	for key, di := range tree {
		fmt.Println(key, "deps:", di.deps)
		//		for _, info := range di.infos {
		//			fmt.Println(info.Name())
		//		}
	}
}

func try2(root string) {
	depLine := func(deps int) string {
		str := ""
		for i := 0; i != deps; i++ {
			str += fmt.Sprintf(" - ")
		}
		return str
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
			if info.IsDir() {
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
	log.SetPrefix("tree:")
	log.SetFlags(log.Lshortfile)

	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("root:", root)

	//try1(root)
	try2(root)
}
