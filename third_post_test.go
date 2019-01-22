package go_code_analysis

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"testing"
)

var Gopath string = ""

func init() {
	path := os.Getenv("GOPATH")
	log.Printf("path:%s", path)
	if path == "" {
		path = "/usr/local/gopath"
	}
	Gopath = path
}

func TestAutoGenContext(t *testing.T) {
	GFixedFunc = make(map[string]Fixed)

	project := "github.com/baixiaoustc/go_code_analysis/example"
	dir := fmt.Sprintf("%s/src/%s", Gopath, project)
	var files []string
	files, err := WalkDir(dir, ".go")
	if err != nil {
		log.Printf("error:%v", err)
		return
	}
	doFind(files)

	for k, v := range GFixedFunc {
		log.Printf("GFixedFunc:%s %v", k, v)
	}

	if len(GFixedFunc) == 0 {
		log.Printf("no need fix func")
		return
	}

	args := []string{project}
	doAnalysis(&build.Default, false, args)

	callMap, err := Analysis.render(project)
	if err != nil {
		log.Printf("error:%v", err)
		return
	}
	for k, v := range callMap {
		log.Printf("正向调用关系:%s %+v", k, v)
	}

	doRelation(callMap)
	for k, v := range GFixedFunc {
		re := CalledRelation{
			Callees: make([]FuncDesc, 0),
		}
		depthTraversal(v.RelationsTree, "", re, &v.RelationList)
		GFixedFunc[k] = v
	}
	for k, v := range GFixedFunc {
		log.Printf("GFixedFunc:%s %v", k, v)
	}
	doFix()
}
