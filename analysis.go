package go_code_analysis

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
	Package string
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
