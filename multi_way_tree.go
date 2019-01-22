package go_code_analysis

import (
	"fmt"
	"log"
)

type MWTNode struct {
	Key      string
	Value    FuncDesc
	N        int
	Children []*MWTNode
}

func BuildFromCallMap(head *MWTNode, callMap map[string]CallerRelation) {
	nodeMap := make(map[string]struct{})
	nodeList := make([]*MWTNode, 1)
	nodeList[0] = head

	for {
		if len(nodeList) == 0 {
			break
		}

		tmp := nodeList[0]
		log.Printf("tmp %+v", tmp)
		for callerName, callRelation := range callMap {
			for _, callee := range callRelation.Callees {
				if tmp.Key == fmt.Sprintf("%s.%s", callee.Package, callee.Name) {
					log.Printf("found caller:%s -> callee:%s", callerName, callee)

					key := fmt.Sprintf("%s.%s", callRelation.Caller.Package, callRelation.Caller.Name)
					if _, ok := nodeMap[key]; !ok {
						newNode := &MWTNode{
							Key:      key,
							Value:    FuncDesc{callRelation.Caller.File, callRelation.Caller.Package, callRelation.Caller.Name},
							Children: make([]*MWTNode, 0),
						}
						tmp.N++
						tmp.Children = append(tmp.Children, newNode)
						nodeList = append(nodeList, newNode)
					} else {
						nodeMap[key] = struct{}{}
					}
				}
			}
		}
		nodeList = nodeList[1:]

		//log.Printf("head %+v", head)
		log.Printf("nodeList len:%d", len(nodeList))
	}
}

func depthTraversal(head *MWTNode, s string, re CalledRelation, list *[]CalledRelation) {
	s = fmt.Sprintf("%s<-%s", s, head.Key)
	re.Callees = append(re.Callees, head.Value)
	//log.Printf("%+v: %s %+v", head, s, re.Callees)

	if head.N == 0 {
		log.Printf("找到反向调用链:%s", s)
		log.Printf("re.Callees:%+v", re.Callees)
		*list = append(*list, re)
		s = ""
		re.Callees = make([]FuncDesc, 0)
	} else {
		for _, node := range head.Children {
			depthTraversal(node, s, re, list)
		}
	}
}
