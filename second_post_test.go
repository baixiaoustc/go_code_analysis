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
