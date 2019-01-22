package go_code_analysis

import (
	"go/build"
	"log"
	"testing"
)

func TestAnalysisCallGraphy(t *testing.T) {
	project := "github.com/baixiaoustc/go_code_analysis/example"
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
}

func TestAnalysisReverceCallGraphy(t *testing.T) {
	project := "github.com/baixiaoustc/go_code_analysis/example"
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

	tree := &MWTNode{
		Key:      "main.test4a",
		Value:    FuncDesc{"/Users/baixiao/Go/src/github.com/baixiaoustc/go_code_analysis/example/test4a.go", "main", "test4a"},
		Children: make([]*MWTNode, 0),
	}
	BuildFromCallMap(tree, callMap)
	re := CalledRelation{
		Callees: make([]FuncDesc, 0),
	}
	list := make([]CalledRelation, 0)
	depthTraversal(tree, "", re, &list)
	for i := range list {
		log.Printf("list%d: %+v", i, list[i])
	}
}
