package go_code_analysis

import (
	"fmt"
	"go/build"
	"log"
	"time"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

//关键函数定义
type Fixed struct {
	FuncDesc
	RelationsTree *MWTNode //反向调用关系，可能有多条调用链到达关键函数
	RelationList  []CalledRelation
	CanFix        bool //能反向找到gin.Context即可以自动修复
}

//函数定义
type FuncDesc struct {
	File    string //文件路径
	Package string //package名
	Name    string //函数名，格式为Package.Func
}

//描述一个函数调用N个函数的一对多关系
type CallerRelation struct {
	Caller  FuncDesc
	Callees []FuncDesc
}

//描述关键函数的一条反向调用关系
type CalledRelation struct {
	Callees []FuncDesc
	CanFix  bool //该调用关系能反向找到gin.Context即可以自动修复
}

var Analysis *analysis

type analysis struct {
	prog   *ssa.Program
	conf   loader.Config
	pkgs   []*ssa.Package
	mains  []*ssa.Package
	result *pointer.Result
}

func doAnalysis(buildCtx *build.Context, tests bool, args []string) {
	t0 := time.Now()
	conf := loader.Config{Build: buildCtx}
	_, err := conf.FromArgs(args, tests)
	if err != nil {
		log.Printf("invalid args:", err)
		return
	}
	load, err := conf.Load()
	if err != nil {
		log.Printf("failed conf load:", err)
		return
	}
	log.Printf("loading.. %d imported (%d created) took: %v",
		len(load.Imported), len(load.Created), time.Since(t0))

	t0 = time.Now()

	prog := ssautil.CreateProgram(load, 0)
	prog.Build()
	pkgs := prog.AllPackages()

	var mains []*ssa.Package
	if tests {
		for _, pkg := range pkgs {
			if main := prog.CreateTestMainPackage(pkg); main != nil {
				mains = append(mains, main)
			}
		}
		if mains == nil {
			log.Fatalln("no tests")
		}
	} else {
		mains = append(mains, ssautil.MainPackages(pkgs)...)
		if len(mains) == 0 {
			log.Printf("no main packages")
		}
	}
	log.Printf("building.. %d packages (%d main) took: %v",
		len(pkgs), len(mains), time.Since(t0))

	t0 = time.Now()
	ptrcfg := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(ptrcfg)
	if err != nil {
		log.Fatalln("analyze failed:", err)
	}
	log.Printf("analysis took: %v", time.Since(t0))

	Analysis = &analysis{
		prog:   prog,
		conf:   conf,
		pkgs:   pkgs,
		mains:  mains,
		result: result,
	}
}

type renderOpts struct {
	focus   string
	group   []string
	ignore  []string
	include []string
	limit   []string
	nointer bool
	nostd   bool
}

func (a *analysis) render(project string) (map[string]CallerRelation, error) {
	var err error
	var focusPkg *build.Package
	opts := renderOpts{
		//focus: focus,
		//group:  []string{"controller"},
		//ignore: []string{"third", "backend/common", fmt.Sprintf("%s/vendor", project)},
		//include: []string{"backend/code_inspector/testing_bai"},
		//limit: []string{"backend/code_inspector"},
		//nointer: nointer,
		nostd: true,
	}

	callMap, err := printOutput(a.prog, a.mains[0].Pkg, a.result.CallGraph,
		focusPkg, opts.limit, opts.ignore, opts.include, opts.group, opts.nostd, opts.nointer)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %v", err)
	}

	return callMap, nil
}
