package script

import (
	"fmt"
	"strings"
)

// A Function accepts inputs, executes a set of Nodes, and returns a single output
type Function struct {
	Identifier string
	In         Declarations
	Out        *Variable
	Address    uint32
	Decls      Declarations
	instrs     *Instructions

	*BasicBlock
	blocks        map[uint32]*BasicBlock
	blocksVisited map[uint32]bool
}

func NewFunction() *Function {
	fn := &Function{}
	block := &BasicBlock{
		ParentFunc: fn,
		instrs:     &Instructions{},
	}
	fn.BasicBlock = block
	fn.blocks = make(map[uint32]*BasicBlock)
	fn.blocksVisited = make(map[uint32]bool)
	fn.instrs = &Instructions{}
	return fn
}

func (fn *Function) CString() string {
	args := make([]string, len(fn.In.Vars))
	for i, arg := range fn.In.Vars {
		args[i] = arg.CString()
	}

	return fmt.Sprintf("%v %v(%v) {\n%v\n%v}", fn.Out.Type.CString(), fn.Identifier, strings.Join(args, ", "), fn.Decls.CString(), fn.BasicBlock.CString())
}

func (fn *Function) resetBlocksVisited() {
	fn.blocksVisited = make(map[uint32]bool)
}

func (fn *Function) inferReturnType(ret Instruction) {
	retVar := fn.Out
	op := ret.Operands.(*RetOperands)
	if op.NumReturnVals == 0 {
		retVar.InferType(VoidType)
	} else if op.NumReturnVals == 1 {
		retVal := fn.peekNode()
		retVar.InferType(retVal.(DataTypeable).DataType())
	} else {
		panic("unable to infer return value of function")
	}
}

type BasicBlock struct {
	Statements []Node
	ParentFunc *Function

	instrs *Instructions

	nodeStack *link
	Ins, Outs []*BasicBlock
}

func newBlock(parent *Function) *BasicBlock {
	return &BasicBlock{
		ParentFunc: parent,
		instrs: &Instructions{
			code: make([]InstructionState, 0),
		},
	}
}

func (b *BasicBlock) Empty() bool {
	return len(b.instrs.code) == 0
}

func (b *BasicBlock) StartAddress() uint32 {
	if len(b.instrs.code) > 0 {
		return b.instrs.code[0].Address
	}
	return 0
}

func (b *BasicBlock) VariableByName(identifier string) *Variable {
	return b.ParentFunc.Decls.VariableByName(identifier)
}

func (b *BasicBlock) VariableByIndex(index int) *Variable {
	return b.ParentFunc.Decls.VariableByIndex(index)
}

func (block *BasicBlock) emitStatement(stmt Node) {
	block.Statements = append(block.Statements, stmt)
}

func (block *BasicBlock) emitComment(format string, args ...interface{}) {
	commentStr := fmt.Sprintf(format, args...)
	comment := Comment(commentStr)
	block.emitStatement(comment)
	fmt.Println(commentStr)
}

func (block *BasicBlock) pushNode(node Node) {
	block.nodeStack = &link{
		Node: node,
		next: block.nodeStack,
	}
	//block.emitComment("pushing %#v at stack idx %v", node, block.nodeStackIdx)
}

func (block *BasicBlock) popNode() Node {
	popped := block.nodeStack
	if popped == nil {
		fmt.Println("node stack underflow")
		return Immediate{&Immediate32Operands{Val: 0xBABE}}
	}

	block.nodeStack = popped.next
	return popped.Node
}

func (block *BasicBlock) peekNode() Node {
	popped := block.nodeStack
	if popped == nil {
		//		fmt.Printf("peek %v\n", block.nodeStackIdx)
		fmt.Println("node stack underflow")
		return Immediate{&Immediate32Operands{Val: 0xBABE}}
	}

	return popped.Node
}

func (block *BasicBlock) nextInstruction() Instruction {
	prevState := block.instrs.prevInstructionState()
	if prevState != nil {
		prevState.nodeStack = block.nodeStack
	}

	return block.instrs.nextInstruction()
}

func (block *BasicBlock) peekInstruction() Instruction {
	return block.instrs.peekInstruction()
}

func (block *BasicBlock) CString() string {
	stmts := make([]string, len(block.Statements))
	for i, s := range block.Statements {

		hasSemicolon := false
		switch s.(type) {
		case IfStmt:
		case Comment:

		default:
			hasSemicolon = true
		}

		stmt := s.CString()
		if hasSemicolon {
			fmt.Sprintf("%v;", stmt)
		}

		stmts[i] = fmt.Sprintf("\t%v\n", stmt)
	}

	return strings.Join(stmts, "")
}
