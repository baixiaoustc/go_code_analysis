package go_code_analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
)

var GFset *token.FileSet
var GFixedFunc map[string]Fixed //key的格式为Package.Func

func stmtCase(stmt ast.Stmt, todo func(call *ast.CallExpr) bool) bool {
	switch t := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := t.X.(*ast.CallExpr); ok {
			return todo(call)
		}
	case *ast.ReturnStmt:
		for i, p := range t.Results {
			log.Printf("return表达式%d:%v at line:%v", i, p, GFset.Position(p.Pos()))
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
						log.Printf("构造赋值%d:%+v at line:%v", i, t.Value, GFset.Position(p.Pos()))
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

func (f *FindContext) Visit(n ast.Node) (w ast.Visitor) {
	if n == nil {
		return f
	}

	if fn, ok := n.(*ast.FuncDecl); ok {
		log.Printf("函数[%s.%s]开始 at line:%v", f.Package, fn.Name.Name, GFset.Position(fn.Pos()))
		f.LocalFunc = fn
	} else {
		log.Printf("类型%T at line:%v", n, GFset.Position(n.Pos()))
	}

	find := AllCallCase(n, f.FindCallFunc)

	if find {
		name := fmt.Sprintf("%s.%s", f.Package, f.LocalFunc.Name)
		GFixedFunc[name] = Fixed{FuncDesc: FuncDesc{f.File, f.Package, f.LocalFunc.Name.Name}}
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
