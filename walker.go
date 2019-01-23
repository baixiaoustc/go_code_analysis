package go_code_analysis

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var GFset *token.FileSet        //全局存储token的position
var GFixedFunc map[string]Fixed //key的格式为Package.Func

func stmtCase(stmt ast.Stmt, todo func(call *ast.CallExpr) bool) bool {
	switch t := stmt.(type) {
	case *ast.ExprStmt:
		log.Printf("表达式语句%+v at line:%v", t, GFset.Position(t.Pos()))
		if call, ok := t.X.(*ast.CallExpr); ok {
			return todo(call)
		}
	case *ast.ReturnStmt:
		for i, p := range t.Results {
			log.Printf("return语句%d:%v at line:%v", i, p, GFset.Position(p.Pos()))
			if call, ok := p.(*ast.CallExpr); ok {
				return todo(call)
			}
		}
	case *ast.AssignStmt:
		//函数体里的构造类型 999
		for _, p := range t.Rhs {
			switch t := p.(type) {
			case *ast.CompositeLit:
				for i, p := range t.Elts {
					switch t := p.(type) {
					case *ast.KeyValueExpr:
						log.Printf("构造赋值语句%d:%+v at line:%v", i, t.Value, GFset.Position(p.Pos()))
						if call, ok := t.Value.(*ast.CallExpr); ok {
							return todo(call)
						}
					}
				}
			}
		}
	default:
		log.Printf("不匹配的类型:%T", stmt)
	}
	return false
}

//调用函数的N种情况
//对函数调用使用todo适配，并返回是否适配成功
func AllCallCase(n ast.Node, todo func(call *ast.CallExpr) bool) (find bool) {

	//函数体里的直接调用 000
	if fn, ok := n.(*ast.FuncDecl); ok {
		for i, p := range fn.Body.List {
			log.Printf("函数体表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
			find = find || stmtCase(p, todo)
		}

		log.Printf("func:%+v done", fn.Name.Name)
	}

	//if语句里
	if ifstmt, ok := n.(*ast.IfStmt); ok {
		log.Printf("if语句开始:%T %+v", ifstmt, GFset.Position(ifstmt.If))

		//if的赋值表达式 111
		if a, ok := ifstmt.Init.(*ast.AssignStmt); ok {
			for i, p := range a.Rhs {
				log.Printf("if语句赋值%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
				switch call := p.(type) {
				case *ast.CallExpr:
					c := todo(call)
					find = find || c
				}
			}
		}

		//if的花括号里面 222
		for i, p := range ifstmt.Body.List {
			log.Printf("if语句内部表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
			c := stmtCase(p, todo)
			find = find || c
		}

		//if的else里面 333
		if b, ok := ifstmt.Else.(*ast.BlockStmt); ok {
			for i, p := range b.List {
				log.Printf("if语句else表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
				c := stmtCase(p, todo)
				find = find || c
			}
		}

		log.Printf("if语句结束:%+v done", GFset.Position(ifstmt.End()))
	}

	//赋值语句 444
	if assign, ok := n.(*ast.AssignStmt); ok {
		log.Printf("赋值语句开始:%T %s", assign, GFset.Position(assign.Pos()))
		for i, p := range assign.Rhs {
			log.Printf("赋值表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
			switch t := p.(type) {
			case *ast.CallExpr:
				c := todo(t)
				find = find || c
			case *ast.CompositeLit:
				//构造表达式 999
				for i, p := range t.Elts {
					switch t := p.(type) {
					case *ast.KeyValueExpr:
						log.Printf("构造赋值%d:%+v at line:%v", i, t.Value, GFset.Position(p.Pos()))
						if call, ok := t.Value.(*ast.CallExpr); ok {
							c := todo(call)
							find = find || c
						}
					}
				}
			}
		}
	}

	if gostmt, ok := n.(*ast.GoStmt); ok {
		log.Printf("go语句开始:%T %s", gostmt.Call.Fun, GFset.Position(gostmt.Go))

		//go后面直接调用 555
		c := todo(gostmt.Call)
		find = find || c

		//go func里面的调用 666
		if g, ok := gostmt.Call.Fun.(*ast.FuncLit); ok {
			for i, p := range g.Body.List {
				log.Printf("go语句表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
				c := stmtCase(p, todo)
				find = find || c
			}
		}

		log.Printf("go语句结束:%+v done", GFset.Position(gostmt.Go))
	}

	if deferstmt, ok := n.(*ast.DeferStmt); ok {
		log.Printf("defer语句开始:%T %s", deferstmt.Call.Fun, GFset.Position(deferstmt.Defer))

		//defer后面直接调用 777
		c := todo(deferstmt.Call)
		find = find || c

		//defer func里面的调用 888
		if g, ok := deferstmt.Call.Fun.(*ast.FuncLit); ok {
			for i, p := range g.Body.List {
				log.Printf("defer语句内部表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
				c := stmtCase(p, todo)
				find = find || c
			}
		}

		log.Printf("defer语句结束:%+v done", GFset.Position(deferstmt.Defer))
	}

	if fostmt, ok := n.(*ast.ForStmt); ok {
		//for语句对应aaa和bbb
		log.Printf("for语句开始:%T %s", fostmt.Body, GFset.Position(fostmt.Pos()))
		for i, p := range fostmt.Body.List {
			log.Printf("for语句函数体表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
			c := stmtCase(p, todo)
			find = find || c
		}
	}

	if rangestmt, ok := n.(*ast.RangeStmt); ok {
		//range语句对应ccc
		log.Printf("range语句开始:%T %s", rangestmt.Body, GFset.Position(rangestmt.Pos()))
		for i, p := range rangestmt.Body.List {
			log.Printf("range语句函数体表达式%d:%T at line:%v", i, p, GFset.Position(p.Pos()))
			c := stmtCase(p, todo)
			find = find || c
		}
	}

	return
}

type FindContext struct {
	File      string
	Package   string
	LocalFunc *ast.FuncDecl
}

func (f *FindContext) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return f
	}
	var FuncType *ast.Ident //用来存储函数是不是某个类的方法

	if fn, ok := n.(*ast.FuncDecl); ok {
		log.Printf("函数[%s.%s]开始 at line:%v", f.Package, fn.Name.Name, GFset.Position(fn.Pos()))
		f.LocalFunc = fn

		if fn.Recv != nil && len(fn.Recv.List) == 1 {
			FuncType = fn.Recv.List[0].Type.(*ast.Ident)
		}
	} else {
		log.Printf("类型%T at line:%v", n, GFset.Position(n.Pos()))
	}

	find := AllCallCase(n, f.FindCallFunc)

	if find {
		if FuncType != nil {
			name := fmt.Sprintf("%s.%s@%s", f.Package, FuncType.Name, f.LocalFunc.Name.Name)
			GFixedFunc[name] = Fixed{FuncDesc: FuncDesc{f.File, f.Package, fmt.Sprintf("%s@%s", FuncType.Name, f.LocalFunc.Name.Name)}}
		} else {
			name := fmt.Sprintf("%s.%s", f.Package, f.LocalFunc.Name.Name)
			GFixedFunc[name] = Fixed{FuncDesc: FuncDesc{f.File, f.Package, f.LocalFunc.Name.Name}}
		}
	}

	return f
}

func (f *FindContext) FindCallFunc(call *ast.CallExpr) bool {
	if call == nil {
		return false
	}

	log.Printf("call func:%+v, %v", call.Fun, call.Args)

	if callFunc, ok := call.Fun.(*ast.SelectorExpr); ok {
		if fmt.Sprint(callFunc.X) == "context" && fmt.Sprint(callFunc.Sel) == "WithCancel" {
			if len(call.Args) > 0 {
				if argu, ok := call.Args[0].(*ast.Ident); ok {
					log.Printf("argu type:%T, %s", argu.Name, argu.String())
					if argu.Name == "nil" {
						location := fmt.Sprint(GFset.Position(argu.NamePos))
						log.Printf("找到关键函数:%s.%s at line:%v", callFunc.X, callFunc.Sel, location)
						return true
					}
				}
			}
		}
	}

	return false
}

//主要用于将调用链里面的nil替换为ctx
//并判断填充父函数的行参context.Context
//并在源头函数生成Context的起点
type FixContext struct {
	Type       GenFuncType
	File       string
	Package    string
	LocalFunc  *ast.FuncDecl
	TargetFunc FuncDesc //希望自动修复的函数
	CalleeFunc FuncDesc //上述函数调用的下一级函数
}

type GenFuncType int

const (
	KeyFunc GenFuncType = iota
	TransFunc
	SourceFunc
)

func (f *FixContext) Visit(n ast.Node) (w ast.Visitor) {
	//time.Sleep(10 * time.Millisecond)
	if n == nil {
		return f
	}
	var FuncType *ast.Ident //用来存储函数是不是某个类的方法

	if fn, ok := n.(*ast.FuncDecl); ok {
		log.Printf("函数[%s.%s]开始 at line:%v", f.Package, fn.Name.Name, GFset.Position(fn.Pos()))
		f.LocalFunc = fn

		if fn.Recv != nil && len(fn.Recv.List) == 1 {
			FuncType = fn.Recv.List[0].Type.(*ast.Ident)
		}
	}

	if f.LocalFunc != nil {
		if FuncType != nil {
			name := fmt.Sprintf("%s@%s", FuncType.Name, f.LocalFunc.Name.Name)
			if name != f.TargetFunc.Name {
				//log.Printf("不匹配1 %s,%s", name, f.TargetFunc.Name)
				return f
			}
		} else {
			name := fmt.Sprintf("%s", f.LocalFunc.Name.Name)
			if name != f.TargetFunc.Name {
				//log.Printf("不匹配2 %s,%s", name, f.TargetFunc.Name)
				return f
			}
		}
	}

	find := AllCallCase(n, f.FixCallFunc)

	if find {
		//如果关键函数本身就是源头函数，则不需要往上递归了
		if fmt.Sprintf("%s.%s", f.Package, f.LocalFunc.Name.Name) == "main.main" {
			log.Printf("函数[%s.%s]不需要往上递归 at line:%v", f.Package, f.LocalFunc.Name.Name, GFset.Position(f.LocalFunc.Pos()))
			//尝试直接生成ctx
			f.genSourceCtx(f.LocalFunc)
		} else {
			//如果修改了函数里面的nil为ctx，需要判断本函数的入参列表有没有ctx context.Context
			log.Printf("函数[%s.%s]内部修复 at line:%v", f.Package, f.LocalFunc.Name, GFset.Position(f.LocalFunc.Pos()))
			f.FixLocalunc()
		}
	}

	return f
}

//如果是关键函数：
//查看关键函数内部是否是nil值填充的调用链，如果是，就将nil替换为ctx
//如果是传递函数或者源头函数：
//在调用处插入第一个ctx参数，区分跨package和本package两种情况
func (f *FixContext) FixCallFunc(call *ast.CallExpr) bool {
	if call == nil {
		return false
	}

	log.Printf("call func:%+v, %v", call.Fun, call.Args)

	if f.Type == KeyFunc {
		if callFunc, ok := call.Fun.(*ast.SelectorExpr); ok {
			//找到关键函数调用
			if fmt.Sprint(callFunc.X) == "context" && fmt.Sprint(callFunc.Sel) == "WithCancel" {
				log.Printf("关键函数调用 %s, %s", callFunc.X, callFunc.Sel)
				return f.replaceNilToCtx(call)
			}
		}
	} else if f.Type == TransFunc || f.Type == SourceFunc {
		if callPackage, ok := call.Fun.(*ast.SelectorExpr); ok {
			//跨package调用
			if fmt.Sprint(callPackage.X) == f.CalleeFunc.Package && fmt.Sprint(callPackage.Sel) == f.CalleeFunc.Name {
				log.Printf("跨package调用 %+v, %+v", callPackage, f.CalleeFunc)
				return f.insertCtxInBody(call)
			}
			//也有可能是本package的类的函数
			if list := strings.Split(f.CalleeFunc.Name, "@"); len(list) == 2 {
				if fmt.Sprint(callPackage.Sel) == list[1] {
					log.Printf("本package调用类的函数 %+v, %+v", callPackage, f.CalleeFunc)
					return f.insertCtxInBody(call)
				}
			}
		}
		if callDirect, ok := call.Fun.(*ast.Ident); ok {
			//本package调用
			if callDirect.Name == f.CalleeFunc.Name {
				log.Printf("本package调用 %+v, %T", call.Fun, call.Fun)
				return f.insertCtxInBody(call)
			}
		}
	}

	return false
}

//关键函数，函数体的调用关系处将实参nil改为ctx
func (f *FixContext) replaceNilToCtx(call *ast.CallExpr) bool {
	if len(call.Args) > 0 {
		//log.Printf("argu type:%T", call.Args[0])
		if argum, ok := call.Args[0].(*ast.Ident); ok {
			log.Printf("argu type:%T, %s, %v", argum.Name, argum.String(), argum.NamePos)
			if argum.Name == "nil" {
				location := fmt.Sprint(GFset.Position(argum.NamePos))
				log.Printf("here at %s", location)

				//把「nil」改为「ctx」
				call.Args[0].(*ast.Ident).Name = "ctx"
				log.Printf("函数[%s.%s]替换ctx成功", f.Package, f.LocalFunc.Name.Name)
				return true
			}
		}
	}

	return false
}

//函数体的调用关系处实参插一个：ctx
func (f *FixContext) insertCtxInBody(call *ast.CallExpr) bool {
	if len(call.Args) > 0 {
		if argum, ok := call.Args[0].(*ast.Ident); ok {
			log.Printf("argu type:%T, %s, %v, %+v", argum.Name, argum.String(), argum.NamePos, argum.Obj)
			if argum.Name == "ctx" {
				log.Printf("函数[%s.%s] already have context in argument", f.Package, call.Fun)
				return false
			}
			//也有可能之前在传递过程中是nil
			if argum.Name == "nil" {
				//把「nil」改为「ctx」
				call.Args[0].(*ast.Ident).Name = "ctx"
				log.Printf("函数[%s.%s]替换ctx成功", f.Package, f.LocalFunc.Name.Name)
				return true
			}
		}
	}

	argums := make([]ast.Expr, len(call.Args)+1)

	name := ast.Ident{
		Name:    "ctx",
		Obj:     ast.NewObj(ast.Var, "ctx"),
		NamePos: call.Pos() + 1}
	argums[0] = &name
	log.Printf("函数[%s.%s] 构造argum: %+v", f.Package, f.LocalFunc.Name.Name, argums[0])

	for i := 0; i < len(call.Args); i++ {
		argums[i+1] = call.Args[i]
	}
	call.Args = argums

	return true
}

//关键函数和传递函数如果没有context，需要加在第一个入参，并记录后递归向上查询
//源头函数如果没有产生ctx，则需要自动生成一个
func (f *FixContext) FixLocalunc() {
	fn := f.LocalFunc

	//看看本函数是不是已经有了ctx，如果有就退出
	for _, d := range fn.Body.List {
		if assign, ok := d.(*ast.AssignStmt); ok {
			for _, a := range assign.Lhs {
				if v, ok := a.(*ast.Ident); ok {
					if v.Name == "ctx" {
						return
					}
				}
			}
		}
	}

	if f.Type == KeyFunc || f.Type == TransFunc {
		f.insertCtxInParam(fn)
	} else if f.Type == SourceFunc {
		f.genSourceCtx(fn)
	}
}

//函数行参插一个：ctx context.Context
func (f *FixContext) insertCtxInParam(fn *ast.FuncDecl) {
	if len(fn.Type.Params.List) > 0 {
		param0 := fn.Type.Params.List[0]
		log.Printf("本函数[%s.%s] param0: %+v, type:%+v", f.Package, fn.Name, param0, param0.Type)
		if param0.Names[0].Name == "ctx" {
			log.Printf("本函数[%s.%s] already have context in param", f.Package, fn.Name)
			return
		}
	}

	params := make([]*ast.Field, len(fn.Type.Params.List)+1)

	names := &ast.Ident{
		Name:    "ctx",
		Obj:     ast.NewObj(ast.Var, "ctx"),
		NamePos: fn.Body.Pos() + 1}
	types := &ast.Ident{
		Name:    "context.Context",
		NamePos: names.End() + 1}
	params[0] = &ast.Field{
		Names: []*ast.Ident{names},
		Type:  types}
	log.Printf("本函数[%s.%s] 构造param: %+v", f.Package, fn.Name, params[0])
	for i := 0; i < len(fn.Type.Params.List); i++ {
		params[i+1] = fn.Type.Params.List[i]
	}
	fn.Type.Params.List = params
}

//函数体写一行：ctx := context.Background()
func (f *FixContext) genSourceCtx(fn *ast.FuncDecl) {
	for i, stmt := range fn.Body.List {
		log.Printf("%d stmt:%+v", i, stmt)
		if assign, ok := stmt.(*ast.AssignStmt); ok {
			log.Printf("赋值语句开始:%T %s", assign, GFset.Position(assign.Pos()))

			for i, p := range assign.Lhs {
				log.Printf("赋值表达式%d:%s at line:%v", i, p, GFset.Position(p.Pos()))
				if fmt.Sprint(p) == "ctx" {
					log.Printf("本函数[%s.%s] already have context generated", f.Package, fn.Name)
					return
				}
			}
		}
	}

	bodies := make([]ast.Stmt, len(fn.Body.List)+1)

	lhs := ast.Ident{
		Name:    "ctx",
		Obj:     ast.NewObj(ast.Var, "ctx"),
		NamePos: fn.Body.Pos() + 1}

	x := ast.Ident{
		Name:    "context",
		Obj:     ast.NewObj(ast.Var, "context"),
		NamePos: fn.Body.Pos() + 1 + token.Pos(len("ctx := "))}
	sel := ast.Ident{
		Name:    "Background",
		Obj:     ast.NewObj(ast.Var, "Background"),
		NamePos: fn.Body.Pos() + 1 + token.Pos(len("ctx := context."))}
	call := ast.SelectorExpr{
		X:   &x,
		Sel: &sel}

	rhs := ast.CallExpr{
		Fun:    &call,
		Args:   []ast.Expr{},
		Lparen: fn.Body.Pos() + token.Pos(len("ctx := context.Background(")+1),
		Rparen: fn.Body.Pos() + token.Pos(len("ctx := context.Background()")+1)}

	assign := &ast.AssignStmt{
		Lhs:    []ast.Expr{&lhs},
		Rhs:    []ast.Expr{&rhs},
		TokPos: lhs.Pos() + 1,
		Tok:    token.DEFINE}

	bodies[0] = assign
	log.Printf("本函数[%s.%s] 构造stmt: %+v", f.Package, fn.Name, bodies[0])
	for i := 0; i < len(fn.Body.List); i++ {
		bodies[i+1] = fn.Body.List[i]
	}
	fn.Body.List = bodies
}

func doFind(files []string) {
	for _, file := range files {
		// Create the AST by parsing src.
		fset := token.NewFileSet() // positions are relative to fset
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			panic(err)
		} else {
			GFset = fset
		}

		find := &FindContext{
			File:    file,
			Package: f.Name.Name,
		}
		ast.Walk(find, f)
	}
}

func doRelation(callMap map[string]CallerRelation) {
	//填充关键函数的反向调用链关系
	for k, v := range GFixedFunc {
		tree := &MWTNode{
			Key:      k,
			Value:    v.FuncDesc,
			Children: make([]*MWTNode, 0),
		}
		BuildFromCallMap(tree, callMap)

		GFixedFunc[k] = Fixed{
			FuncDesc:      v.FuncDesc,
			RelationsTree: tree,
		}
	}
}

func doFix() {
	for k, v := range GFixedFunc {
		log.Printf("开始修复：%s", k)
		for _, relations := range v.RelationList {
			fixWithContext(v, relations)
		}
	}

	return
}

//能够修复的，通过反向调用链补齐context
func fixWithContext(fixed Fixed, relations CalledRelation) {
	//1，关键函数内部替换
	file := fixed.FuncDesc.File
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		log.Printf("ParseFile %s error:%v", file, err)
		return
	} else {
		GFset = fset
	}

	fix := &FixContext{
		Type:       KeyFunc,
		File:       file,
		Package:    f.Name.Name,
		TargetFunc: FuncDesc{fixed.File, fixed.Package, fixed.Name},
	}
	log.Printf("修复关键函数:%s %+v", fix.TargetFunc.Name, fix)
	ast.Walk(fix, f)

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, f)
	genFile(file, buf)

	//2，传递函数和源头函数处理
	for i, callee := range relations.Callees {
		file := callee.File

		// Create the AST by parsing src.
		fset := token.NewFileSet() // positions are relative to fset
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			log.Printf("ParseFile %s error:%v", file, err)
			return
		} else {
			GFset = fset
		}

		fix := &FixContext{
			Type:       TransFunc,
			File:       file,
			Package:    f.Name.Name,
			TargetFunc: callee,
		}
		if i == 0 {
			fix.CalleeFunc = FuncDesc{fixed.File, fixed.Package, fixed.Name}
		} else {
			fix.CalleeFunc = relations.Callees[i-1]
		}
		if i == len(relations.Callees)-1 {
			fix.Type = SourceFunc
			log.Printf("修复源头函数:%s %+v", fix.TargetFunc.Name, fix)
		} else {
			log.Printf("修复传递函数:%s %+v", fix.TargetFunc.Name, fix)
		}
		ast.Walk(fix, f)

		var buf bytes.Buffer
		printer.Fprint(&buf, fset, f)
		genFile(file, buf)
	}
}

func genFile(file string, buf bytes.Buffer) {
	//替换原文件
	newFile, err := os.Create(file)
	defer newFile.Close()
	if err != nil {
		log.Printf("os.Create %s error:%v", file, err)
		return
	} else {
		newFile.Write(buf.Bytes())
	}

	cmd := fmt.Sprintf("go fmt %s;goimports -w %s", file, file)
	runCmd("/bin/sh", "-c", cmd)
}

func runCmd(name string, args ...string) string {
	// 执行系统命令
	// 第一个参数是命令名称
	// 后面参数可以有多个，命令参数
	cmd := exec.Command(name, args...)
	// 获取输出对象，可以从该对象中读取输出结果
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("%v", err)
		return err.Error()
	}
	// 保证关闭输出流
	defer stderr.Close()
	// 运行命令
	if err := cmd.Start(); err != nil {
		log.Printf("%v", err)
		return err.Error()
	}
	// 读取输出结果
	opBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		log.Printf("%v", err)
		return err.Error()
	}
	log.Printf("%v", string(opBytes))

	//防止进程太多导致：resource temporarily unavailable
	timer := time.AfterFunc(1*time.Second, func() {
		err := cmd.Process.Kill()
		if err != nil {
			//panic(err) // panic as can't kill a process.
			log.Printf("cmd.Process.Kill %v", err)
			return
		}
	})
	err = cmd.Wait()
	if err != nil {
		timer.Stop()
		log.Printf("cmd.Wait %v", err)
		return string(opBytes)
	}
	timer.Stop()
	return string(opBytes)
}
