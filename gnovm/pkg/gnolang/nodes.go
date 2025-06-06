package gnolang

import (
	"fmt"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gnolang/gno/gnovm"
	"go.uber.org/multierr"
)

// ----------------------------------------
// Primitives

type Word int

const (
	// Special words
	ILLEGAL Word = iota

	// Names and basic type literals
	// (these words stand for classes of literals)
	NAME   // main
	INT    // 12345
	FLOAT  // 123.45
	IMAG   // 123.45i
	CHAR   // 'a'
	STRING // "abc"

	// Operators and delimiters
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	BAND     // &
	BOR      // |
	XOR      // ^
	SHL      // <<
	SHR      // >>
	BAND_NOT // &^

	ADD_ASSIGN      // +=
	SUB_ASSIGN      // -=
	MUL_ASSIGN      // *=
	QUO_ASSIGN      // /=
	REM_ASSIGN      // %=
	BAND_ASSIGN     // &=
	BOR_ASSIGN      // |=
	XOR_ASSIGN      // ^=
	SHL_ASSIGN      // <<=
	SHR_ASSIGN      // >>=
	BAND_NOT_ASSIGN // &^=

	LAND  // &&
	LOR   // ||
	ARROW // <-
	INC   // ++
	DEC   // --

	EQL    // ==
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !

	NEQ    // !=
	LEQ    // <=
	GEQ    // >=
	DEFINE // :=

	// Keywords
	BREAK
	CASE
	CHAN
	CONST
	CONTINUE

	DEFAULT
	DEFER
	ELSE
	FALLTHROUGH
	FOR

	FUNC
	GO
	GOTO
	IF
	IMPORT

	INTERFACE
	MAP
	PACKAGE
	RANGE
	RETURN

	SELECT
	STRUCT
	SWITCH
	TYPE
	VAR
)

type Name string

// ----------------------------------------
// Location
// Acts as an identifier for nodes.

type Location struct {
	PkgPath string
	File    string
	Line    int
	Column  int
}

func (loc Location) String() string {
	return fmt.Sprintf("%s/%s:%d:%d",
		loc.PkgPath,
		loc.File,
		loc.Line,
		loc.Column,
	)
}

func (loc Location) IsZero() bool {
	return loc.PkgPath == "" &&
		loc.File == "" &&
		loc.Line == 0 &&
		loc.Column == 0
}

// ----------------------------------------
// Attributes
// All nodes have attributes for general analysis purposes.
// Exported Attribute fields like Loc and Label are persisted
// even after preprocessing.  Temporary attributes (e.g. those
// for preprocessing) are stored in .data.

type GnoAttribute string

const (
	ATTR_PREPROCESSED    GnoAttribute = "ATTR_PREPROCESSED"
	ATTR_PREDEFINED      GnoAttribute = "ATTR_PREDEFINED"
	ATTR_TYPE_VALUE      GnoAttribute = "ATTR_TYPE_VALUE"
	ATTR_TYPEOF_VALUE    GnoAttribute = "ATTR_TYPEOF_VALUE"
	ATTR_IOTA            GnoAttribute = "ATTR_IOTA"
	ATTR_HEAP_DEFINES    GnoAttribute = "ATTR_HEAP_DEFINES" // []Name heap items.
	ATTR_HEAP_USES       GnoAttribute = "ATTR_HEAP_USES"    // []Name heap items used.
	ATTR_SHIFT_RHS       GnoAttribute = "ATTR_SHIFT_RHS"
	ATTR_LAST_BLOCK_STMT GnoAttribute = "ATTR_LAST_BLOCK_STMT"
	ATTR_GLOBAL          GnoAttribute = "ATTR_GLOBAL"
	ATTR_PACKAGE_REF     GnoAttribute = "ATTR_PACKAGE_REF"
	ATTR_PACKAGE_DECL    GnoAttribute = "ATTR_PACKAGE_DECL"
)

type Attributes struct {
	Line   int
	Column int
	Label  Name
	data   map[GnoAttribute]any // not persisted
}

func (attr *Attributes) GetLine() int {
	return attr.Line
}

func (attr *Attributes) SetLine(line int) {
	attr.Line = line
}

func (attr *Attributes) GetColumn() int {
	return attr.Column
}

func (attr *Attributes) SetColumn(column int) {
	attr.Column = column
}

func (attr *Attributes) GetLabel() Name {
	return attr.Label
}

func (attr *Attributes) SetLabel(label Name) {
	attr.Label = label
}

func (attr *Attributes) HasAttribute(key GnoAttribute) bool {
	_, ok := attr.data[key]
	return ok
}

// GnoAttribute must not be user provided / arbitrary,
// otherwise will create potential exploits.
func (attr *Attributes) GetAttribute(key GnoAttribute) any {
	return attr.data[key]
}

func (attr *Attributes) SetAttribute(key GnoAttribute, value any) {
	if attr.data == nil {
		attr.data = make(map[GnoAttribute]any)
	}
	attr.data[key] = value
}

func (attr *Attributes) DelAttribute(key GnoAttribute) {
	if debug && attr.data == nil {
		panic("should not happen, attribute is expected to be non-empty.")
	}
	delete(attr.data, key)
}

// ----------------------------------------
// Node

type Node interface {
	assertNode()
	String() string
	Copy() Node
	GetLine() int
	SetLine(int)
	GetColumn() int
	SetColumn(int)
	GetLabel() Name
	SetLabel(Name)
	HasAttribute(key GnoAttribute) bool
	GetAttribute(key GnoAttribute) any
	SetAttribute(key GnoAttribute, value any)
	DelAttribute(key GnoAttribute)
}

// non-pointer receiver to help make immutable.
func (x *NameExpr) assertNode()          {}
func (x *BasicLitExpr) assertNode()      {}
func (x *BinaryExpr) assertNode()        {}
func (x *CallExpr) assertNode()          {}
func (x *IndexExpr) assertNode()         {}
func (x *SelectorExpr) assertNode()      {}
func (x *SliceExpr) assertNode()         {}
func (x *StarExpr) assertNode()          {}
func (x *RefExpr) assertNode()           {}
func (x *TypeAssertExpr) assertNode()    {}
func (x *UnaryExpr) assertNode()         {}
func (x *CompositeLitExpr) assertNode()  {}
func (x *KeyValueExpr) assertNode()      {}
func (x *FuncLitExpr) assertNode()       {}
func (x *ConstExpr) assertNode()         {}
func (x *FieldTypeExpr) assertNode()     {}
func (x *ArrayTypeExpr) assertNode()     {}
func (x *SliceTypeExpr) assertNode()     {}
func (x *InterfaceTypeExpr) assertNode() {}
func (x *ChanTypeExpr) assertNode()      {}
func (x *FuncTypeExpr) assertNode()      {}
func (x *MapTypeExpr) assertNode()       {}
func (x *StructTypeExpr) assertNode()    {}
func (x *constTypeExpr) assertNode()     {}
func (x *AssignStmt) assertNode()        {}
func (x *BlockStmt) assertNode()         {}
func (x *BranchStmt) assertNode()        {}
func (x *DeclStmt) assertNode()          {}
func (x *DeferStmt) assertNode()         {}
func (x *ExprStmt) assertNode()          {}
func (x *ForStmt) assertNode()           {}
func (x *GoStmt) assertNode()            {}
func (x *IfStmt) assertNode()            {}
func (x *IfCaseStmt) assertNode()        {}
func (x *IncDecStmt) assertNode()        {}
func (x *RangeStmt) assertNode()         {}
func (x *ReturnStmt) assertNode()        {}
func (x *SelectStmt) assertNode()        {}
func (x *SelectCaseStmt) assertNode()    {}
func (x *SendStmt) assertNode()          {}
func (x *SwitchStmt) assertNode()        {}
func (x *SwitchClauseStmt) assertNode()  {}
func (x *EmptyStmt) assertNode()         {}
func (x *bodyStmt) assertNode()          {}
func (x *FuncDecl) assertNode()          {}
func (x *ImportDecl) assertNode()        {}
func (x *ValueDecl) assertNode()         {}
func (x *TypeDecl) assertNode()          {}
func (x *FileNode) assertNode()          {}
func (x *PackageNode) assertNode()       {}

var (
	_ Node = &NameExpr{}
	_ Node = &BasicLitExpr{}
	_ Node = &BinaryExpr{}
	_ Node = &CallExpr{}
	_ Node = &IndexExpr{}
	_ Node = &SelectorExpr{}
	_ Node = &SliceExpr{}
	_ Node = &StarExpr{}
	_ Node = &RefExpr{}
	_ Node = &TypeAssertExpr{}
	_ Node = &UnaryExpr{}
	_ Node = &CompositeLitExpr{}
	_ Node = &KeyValueExpr{}
	_ Node = &FuncLitExpr{}
	_ Node = &ConstExpr{}
	_ Node = &FieldTypeExpr{}
	_ Node = &ArrayTypeExpr{}
	_ Node = &SliceTypeExpr{}
	_ Node = &InterfaceTypeExpr{}
	_ Node = &ChanTypeExpr{}
	_ Node = &FuncTypeExpr{}
	_ Node = &MapTypeExpr{}
	_ Node = &StructTypeExpr{}
	_ Node = &constTypeExpr{}
	_ Node = &AssignStmt{}
	_ Node = &BlockStmt{}
	_ Node = &BranchStmt{}
	_ Node = &DeclStmt{}
	_ Node = &DeferStmt{}
	_ Node = &ExprStmt{}
	_ Node = &ForStmt{}
	_ Node = &GoStmt{}
	_ Node = &IfStmt{}
	_ Node = &IfCaseStmt{}
	_ Node = &IncDecStmt{}
	_ Node = &RangeStmt{}
	_ Node = &ReturnStmt{}
	_ Node = &SelectStmt{}
	_ Node = &SelectCaseStmt{}
	_ Node = &SendStmt{}
	_ Node = &SwitchStmt{}
	_ Node = &SwitchClauseStmt{}
	_ Node = &EmptyStmt{}
	_ Node = &bodyStmt{}
	_ Node = &FuncDecl{}
	_ Node = &ImportDecl{}
	_ Node = &ValueDecl{}
	_ Node = &TypeDecl{}
	_ Node = &FileNode{}
	_ Node = &PackageNode{}
)

// ----------------------------------------
// Expr
//
// expressions generally have no side effects on the caller's context,
// except for channel blocks, type assertions, and panics.

type Expr interface {
	Node
}

type Exprs []Expr

var (
	_ Expr = &NameExpr{}
	_ Expr = &BasicLitExpr{}
	_ Expr = &BinaryExpr{}
	_ Expr = &CallExpr{}
	_ Expr = &IndexExpr{}
	_ Expr = &SelectorExpr{}
	_ Expr = &SliceExpr{}
	_ Expr = &StarExpr{}
	_ Expr = &RefExpr{}
	_ Expr = &TypeAssertExpr{}
	_ Expr = &UnaryExpr{}
	_ Expr = &CompositeLitExpr{}
	_ Expr = &KeyValueExpr{}
	_ Expr = &FuncLitExpr{}
	_ Expr = &ConstExpr{}
)

type NameExprType int

const (
	NameExprTypeNormal      NameExprType = iota // default
	NameExprTypeDefine                          // when defining normally
	NameExprTypeHeapDefine                      // when defining escaped name in loop
	NameExprTypeHeapUse                         // when above used in non-define lhs/rhs
	NameExprTypeHeapClosure                     // when closure captures name
)

type NameExpr struct {
	Attributes
	// TODO rename .Path's to .ValuePaths.
	Path ValuePath // set by preprocessor.
	Name
	Type NameExprType
}

type NameExprs []NameExpr

type BasicLitExpr struct {
	Attributes
	// INT, FLOAT, IMAG, CHAR, or STRING
	Kind Word
	// literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo"
	// or `\m\n\o`
	Value string
}

type BinaryExpr struct { // (Left Op Right)
	Attributes
	Left  Expr // left operand
	Op    Word // operator
	Right Expr // right operand
}

type CallExpr struct { // Func(Args<Varg?...>)
	Attributes
	Func      Expr  // function expression
	Args      Exprs // function arguments, if any.
	Varg      bool  // if true, final arg is variadic.
	NumArgs   int   // len(Args) or len(Args[0].Results)
	WithCross bool  // if called like cross(fn)(...).
}

// if x is *ConstExpr returns its source.
func unwrapConstExpr(x Expr) Expr {
	if cx, ok := x.(*ConstExpr); ok {
		return cx.Source
	}
	return x
}

// returns true if x is of form cross(fn)(...).
func (x *CallExpr) isWithCross() bool {
	if fnc, ok := x.Func.(*CallExpr); ok {
		if nx, ok := unwrapConstExpr(fnc.Func).(*NameExpr); ok {
			if nx.Name == "cross" {
				return true
			}
		}
	}
	return false
}

// returns true if x is of form crossing().
func (x *CallExpr) isCrossing() bool {
	if x == nil {
		return false
	}
	if nx, ok := unwrapConstExpr(x.Func).(*NameExpr); ok {
		if nx.Name == "crossing" {
			return true
		}
	}
	return false
}

func (x *CallExpr) SetWithCross() {
	if !x.isWithCross() {
		panic("expected cross(fn)(...)")
	}
	x.WithCross = true
}

func (x *CallExpr) IsWithCross() bool {
	return x.WithCross
}

type IndexExpr struct { // X[Index]
	Attributes
	X     Expr // expression
	Index Expr // index expression
	HasOK bool // if true, is form: `value, ok := <X>[<Key>]
}

type SelectorExpr struct { // X.Sel
	Attributes
	X    Expr      // expression
	Path ValuePath // set by preprocessor.
	Sel  Name      // field selector
}

type SliceExpr struct { // X[Low:High:Max]
	Attributes
	X    Expr // expression
	Low  Expr // begin of slice range; or nil
	High Expr // end of slice range; or nil
	Max  Expr // maximum capacity of slice; or nil; added in Go 1.2
}

// A StarExpr node represents an expression of the form
// "*" Expression.  Semantically it could be a unary "*"
// expression, or a pointer type.
type StarExpr struct { // *X
	Attributes
	X Expr // operand
}

type RefExpr struct { // &X
	Attributes
	X Expr // operand
}

type TypeAssertExpr struct { // X.(Type)
	Attributes
	X     Expr // expression.
	Type  Expr // asserted type, never nil.
	HasOK bool // if true, is form: `_, ok := <X>.(<Type>)`.
}

// A UnaryExpr node represents a unary expression. Unary
// "*" expressions (dereferencing and pointer-types) are
// represented with StarExpr nodes.  Unary & expressions
// (referencing) are represented with RefExpr nodes.
type UnaryExpr struct { // (Op X)
	Attributes
	X  Expr // operand
	Op Word // operator
}

// MyType{<key>:<value>} struct, array, slice, and map
// expressions.
type CompositeLitExpr struct {
	Attributes
	Type Expr          // literal type; or nil
	Elts KeyValueExprs // list of struct fields; if any
}

// Returns true if any elements are keyed.
// Panics if inconsistent.
func (x *CompositeLitExpr) IsKeyed() bool {
	if len(x.Elts) == 0 {
		return false
	} else if x.Elts[0].Key == nil {
		for i := 1; i < len(x.Elts); i++ {
			if x.Elts[i].Key != nil {
				panic("mixed keyed and unkeyed elements")
			}
		}
		return false
	} else {
		for i := 1; i < len(x.Elts); i++ {
			if x.Elts[i].Key == nil {
				panic("mixed keyed and unkeyed elements")
			}
		}
		return true
	}
}

// A KeyValueExpr represents a single key-value pair in
// struct, array, slice, and map expressions.
type KeyValueExpr struct {
	Attributes
	Key   Expr // or nil
	Value Expr // never nil
}

type KeyValueExprs []KeyValueExpr

// A FuncLitExpr node represents a function literal.  Here one
// can reference statements from an expression, which
// completes the procedural circle.
type FuncLitExpr struct {
	Attributes
	StaticBlock
	Type         FuncTypeExpr // function type
	Body                      // function body
	HeapCaptures NameExprs    // filled in findLoopUses1
}

// The preprocessor replaces const expressions
// with *ConstExpr nodes.
type ConstExpr struct {
	Attributes
	Source Expr // (preprocessed) source of this value.
	TypedValue
}

// ----------------------------------------
// Type(Expressions)
//
// In Go, Type expressions can be evaluated immediately
// without invoking the stack machine.  Exprs in type
// expressions are const (as in array len expr or map key type
// expr) or refer to an exposed symbol (with any pointer
// indirections).  this makes for more optimal performance.
//
// In Gno, type expressions are evaluated on the stack, with
// continuation opcodes, so the Gno VM could support types as
// first class objects.

type TypeExpr interface {
	Expr
	assertTypeExpr()
}

// non-pointer receiver to help make immutable.
func (x *FieldTypeExpr) assertTypeExpr()     {}
func (x *ArrayTypeExpr) assertTypeExpr()     {}
func (x *SliceTypeExpr) assertTypeExpr()     {}
func (x *InterfaceTypeExpr) assertTypeExpr() {}
func (x *ChanTypeExpr) assertTypeExpr()      {}
func (x *FuncTypeExpr) assertTypeExpr()      {}
func (x *MapTypeExpr) assertTypeExpr()       {}
func (x *StructTypeExpr) assertTypeExpr()    {}
func (x *constTypeExpr) assertTypeExpr()     {}

var (
	_ TypeExpr = &FieldTypeExpr{}
	_ TypeExpr = &ArrayTypeExpr{}
	_ TypeExpr = &SliceTypeExpr{}
	_ TypeExpr = &InterfaceTypeExpr{}
	_ TypeExpr = &ChanTypeExpr{}
	_ TypeExpr = &FuncTypeExpr{}
	_ TypeExpr = &MapTypeExpr{}
	_ TypeExpr = &StructTypeExpr{}
	_ TypeExpr = &constTypeExpr{}
)

type FieldTypeExpr struct {
	Attributes
	NameExpr
	Type Expr

	// Currently only BasicLitExpr allowed.
	// NOTE: In Go, only struct fields can have tags.
	Tag Expr
}

type FieldTypeExprs []FieldTypeExpr

// Keep it slow, validating.
// If you need it faster, memoize it elsewhere.
func (ftxz FieldTypeExprs) IsNamed() bool {
	named := false
	for i, ftx := range ftxz {
		if i == 0 {
			if ftx.Name == "" || isMissingResult(ftx.Name) {
				named = false
			} else {
				named = true
			}
		} else {
			if named && (ftx.Name == "" || isMissingResult(ftx.Name)) {
				panic("[]FieldTypeExpr has inconsistent namedness (starts named)")
			} else if !named && (ftx.Name != "" && !isMissingResult(ftx.Name)) {
				panic("[]FieldTypeExpr has inconsistent namedness (starts unnamed)")
			}
		}
	}
	return named
}

type ArrayTypeExpr struct {
	Attributes
	Len Expr // if nil, variadic array lit
	Elt Expr // element type
}

type SliceTypeExpr struct {
	Attributes
	Elt Expr // element type
	Vrd bool // variadic arg expression
}

type InterfaceTypeExpr struct {
	Attributes
	Methods FieldTypeExprs // list of methods
	Generic Name           // for uverse generics
}

type ChanDir int

const (
	SEND ChanDir = 1 << iota
	RECV
)

const (
	BOTH = SEND | RECV
)

type ChanTypeExpr struct {
	Attributes
	Dir   ChanDir // channel direction
	Value Expr    // value type
}

type FuncTypeExpr struct {
	Attributes
	Params  FieldTypeExprs // (incoming) parameters, if any.
	Results FieldTypeExprs // (outgoing) results, if any.
}

type MapTypeExpr struct {
	Attributes
	Key   Expr // const
	Value Expr // value type
}

type StructTypeExpr struct {
	Attributes
	Fields FieldTypeExprs // list of field declarations
}

// Like ConstExpr but for types.
type constTypeExpr struct {
	Attributes
	Source Expr
	Type   Type
}

// ----------------------------------------
// Stmt
//
// statements generally have side effects on the calling context.

type Stmt interface {
	Node
	assertStmt()
}

type Body []Stmt

func (ss Body) GetBody() Body {
	return ss
}

func (ss *Body) SetBody(nb Body) {
	*ss = nb
}

func (ss Body) GetLabeledStmt(label Name) (stmt Stmt, idx int) {
	for idx, stmt = range ss {
		if label == stmt.GetLabel() {
			return stmt, idx
		}
	}
	return nil, -1
}

// Convenience, returns true if first statement is crossing()
func (ss Body) IsCrossing() bool {
	return ss.isCrossing()
}

// XXX deprecate
func (ss Body) isCrossing() bool {
	if len(ss) == 0 {
		return false
	}
	fs := ss[0]
	xs, ok := fs.(*ExprStmt)
	if !ok {
		return false
	}
	cx, ok := xs.X.(*CallExpr)
	return cx.isCrossing()
}

// ----------------------------------------

// non-pointer receiver to help make immutable.
func (*AssignStmt) assertStmt()       {}
func (*BlockStmt) assertStmt()        {}
func (*BranchStmt) assertStmt()       {}
func (*DeclStmt) assertStmt()         {}
func (*DeferStmt) assertStmt()        {}
func (*EmptyStmt) assertStmt()        {} // useful for _ctif
func (*ExprStmt) assertStmt()         {}
func (*ForStmt) assertStmt()          {}
func (*GoStmt) assertStmt()           {}
func (*IfStmt) assertStmt()           {}
func (*IfCaseStmt) assertStmt()       {}
func (*IncDecStmt) assertStmt()       {}
func (*RangeStmt) assertStmt()        {}
func (*ReturnStmt) assertStmt()       {}
func (*SelectStmt) assertStmt()       {}
func (*SelectCaseStmt) assertStmt()   {}
func (*SendStmt) assertStmt()         {}
func (*SwitchStmt) assertStmt()       {}
func (*SwitchClauseStmt) assertStmt() {}
func (*bodyStmt) assertStmt()         {}

var (
	_ Stmt = &AssignStmt{}
	_ Stmt = &BlockStmt{}
	_ Stmt = &BranchStmt{}
	_ Stmt = &DeclStmt{}
	_ Stmt = &DeferStmt{}
	_ Stmt = &EmptyStmt{}
	_ Stmt = &ExprStmt{}
	_ Stmt = &ForStmt{}
	_ Stmt = &GoStmt{}
	_ Stmt = &IfStmt{}
	_ Stmt = &IfCaseStmt{}
	_ Stmt = &IncDecStmt{}
	_ Stmt = &RangeStmt{}
	_ Stmt = &ReturnStmt{}
	_ Stmt = &SelectStmt{}
	_ Stmt = &SelectCaseStmt{}
	_ Stmt = &SendStmt{}
	_ Stmt = &SwitchStmt{}
	_ Stmt = &SwitchClauseStmt{}
	_ Stmt = &bodyStmt{}
)

type AssignStmt struct {
	Attributes
	Lhs Exprs
	Op  Word // assignment word (DEFINE, ASSIGN)
	Rhs Exprs
}

type BlockStmt struct {
	Attributes
	StaticBlock
	Body
}

type BranchStmt struct {
	Attributes
	Op        Word  // keyword word (BREAK, CONTINUE, GOTO, FALLTHROUGH)
	Label     Name  // label name; or empty
	Depth     uint8 // blocks to pop
	BodyIndex int   // index of statement of body
}

type DeclStmt struct {
	Attributes
	Body // (simple) ValueDecl or TypeDecl
}

type DeferStmt struct {
	Attributes
	Call CallExpr
}

// A compile artifact to use in place of nil.
// For example, _ctif() may return an empty statement.
type EmptyStmt struct {
	Attributes
}

type ExprStmt struct {
	Attributes
	X Expr
}

type ForStmt struct {
	Attributes
	StaticBlock
	Init Stmt // initialization (simple) statement; or nil
	Cond Expr // condition; or nil
	Post Stmt // post iteration (simple) statement; or nil
	Body
}

type GoStmt struct {
	Attributes
	Call CallExpr
}

// NOTE: syntactically, code may choose to chain if-else statements
// with `} else if ... {` constructions, but this is not represented
// in the logical AST.
type IfStmt struct {
	Attributes
	StaticBlock
	Init Stmt       // initialization (simple) statement; or nil
	Cond Expr       // condition; or nil
	Then IfCaseStmt // body statements
	Else IfCaseStmt // else statements
}

type IfCaseStmt struct {
	Attributes
	StaticBlock
	Body
}

type IncDecStmt struct {
	Attributes
	X  Expr
	Op Word // INC or DEC
}

type RangeStmt struct {
	Attributes
	StaticBlock
	X          Expr // value to range over
	Key, Value Expr // Key, Value may be nil
	Op         Word // ASSIGN or DEFINE
	Body
	IsMap      bool // if X is map type
	IsString   bool // if X is string type
	IsArrayPtr bool // if X is array-pointer type
}

type ReturnStmt struct {
	Attributes
	Results     Exprs // result expressions; or nil
	CopyResults bool  // copy results to block first
}

type SelectStmt struct {
	Attributes
	Cases []SelectCaseStmt
}

type SelectCaseStmt struct {
	Attributes
	StaticBlock
	Comm Stmt // send or receive statement; nil means default case
	Body
}

type SendStmt struct {
	Attributes
	Chan  Expr
	Value Expr
}

// type ReceiveStmt
// is just AssignStmt with a Receive unary expression.

type SwitchStmt struct {
	Attributes
	StaticBlock
	Init         Stmt               // init (simple) stmt; or nil
	X            Expr               // tag or _.(type) expr; or nil
	IsTypeSwitch bool               // true iff X is .(type) expr
	Clauses      []SwitchClauseStmt // case clauses
	VarName      Name               // type-switched value; or ""
}

type SwitchClauseStmt struct {
	Attributes
	StaticBlock
	Cases Exprs // list of expressions or types; nil means default case
	Body
}

// ----------------------------------------
// bodyStmt (persistent)

// NOTE: embedded in Block.
type bodyStmt struct {
	Attributes
	Body                       // for non-loop stmts
	BodyLen       int          // for for-continue
	NextBodyIndex int          // init:-2, cond/elem:-1, body:0..., post:n
	NumOps        int          // number of Ops, for goto
	NumValues     int          // number of Values, for goto
	NumExprs      int          // number of Exprs, for goto
	NumStmts      int          // number of Stmts, for goto
	Cond          Expr         // for ForStmt
	Post          Stmt         // for ForStmt
	Active        Stmt         // for PopStmt()
	Key           Expr         // for RangeStmt
	Value         Expr         // for RangeStmt
	Op            Word         // for RangeStmt
	ListLen       int          // for RangeStmt only
	ListIndex     int          // for RangeStmt only
	NextItem      *MapListItem // fpr RangeStmt w/ maps only
	StrLen        int          // for RangeStmt w/ strings only
	StrIndex      int          // for RangeStmt w/ strings only
	NextRune      rune         // for RangeStmt w/ strings only
}

func (x *bodyStmt) PopActiveStmt() (as Stmt) {
	as = x.Active
	x.Active = nil
	return
}

func (x *bodyStmt) LastStmt() Stmt {
	return x.Body[x.NextBodyIndex-1]
}

func (x *bodyStmt) String() string {
	next := ""
	if x.NextBodyIndex < 0 {
		next = "(init)"
	} else if x.NextBodyIndex == len(x.Body) {
		next = "(end)"
	} else {
		next = x.Body[x.NextBodyIndex].String()
	}
	active := ""
	if x.Active != nil {
		if x.NextBodyIndex < 0 || x.NextBodyIndex == len(x.Body) {
			// none
		} else if x.Body[x.NextBodyIndex-1] == x.Active {
			active = "*"
		} else {
			active = fmt.Sprintf(" unexpected active: %v", x.Active)
		}
	}
	return fmt.Sprintf("bodyStmt[%d/%d/%d]=%s%s Active:%v",
		x.ListLen,
		x.ListIndex,
		x.NextBodyIndex,
		next,
		active,
		x.Active)
}

// ----------------------------------------
// Simple Statement
// NOTE: SimpleStmt is not used in nodes due to itable conversion costs.
//
// These are used in if, switch, and for statements for simple
// initialization.  The only allowed types are EmptyStmt, ExprStmt,
// SendStmt, IncDecStmt, and AssignStmt.

type SimpleStmt interface {
	Stmt
	assertSimpleStmt()
}

// non-pointer receiver to help make immutable.
func (*EmptyStmt) assertSimpleStmt()  {}
func (*ExprStmt) assertSimpleStmt()   {}
func (*SendStmt) assertSimpleStmt()   {}
func (*IncDecStmt) assertSimpleStmt() {}
func (*AssignStmt) assertSimpleStmt() {}

// ----------------------------------------
// Decl

type Decl interface {
	Node
	GetDeclNames() []Name
	assertDecl()
}

type Decls []Decl

// non-pointer receiver to help make immutable.
func (x *FuncDecl) assertDecl()   {}
func (x *ImportDecl) assertDecl() {}
func (x *ValueDecl) assertDecl()  {}
func (x *TypeDecl) assertDecl()   {}

var (
	_ Decl = &FuncDecl{}
	_ Decl = &ImportDecl{}
	_ Decl = &ValueDecl{}
	_ Decl = &TypeDecl{}
)

type FuncDecl struct {
	Attributes
	StaticBlock
	NameExpr
	IsMethod bool
	Recv     FieldTypeExpr // receiver (if method); or empty (if function)
	Type     FuncTypeExpr  // function signature: parameters and results
	Body                   // function body; or empty for external (non-Go) function

	unboundType *FuncTypeExpr // memoized
}

func (x *FuncDecl) GetDeclNames() []Name {
	if x.IsMethod {
		return nil
	} else {
		return []Name{x.NameExpr.Name}
	}
}

// If FuncDecl is for method, construct a FuncTypeExpr with receiver as first
// parameter.
func (x *FuncDecl) GetUnboundTypeExpr() *FuncTypeExpr {
	if x.IsMethod {
		if x.unboundType == nil {
			x.unboundType = &FuncTypeExpr{
				Attributes: x.Type.Attributes,
				Params:     append([]FieldTypeExpr{x.Recv}, x.Type.Params...),
				Results:    x.Type.Results,
			}
		}
		return x.unboundType
	}
	return &x.Type
}

type ImportDecl struct {
	Attributes
	NameExpr // local package name. required.
	PkgPath  string
}

func (x *ImportDecl) GetDeclNames() []Name {
	if x.NameExpr.Name == "." {
		return nil // ignore
	} else {
		return []Name{x.NameExpr.Name}
	}
}

type ValueDecl struct {
	Attributes
	NameExprs
	Type   Expr  // value type; or nil
	Values Exprs // initial value; or nil (unless const).
	Const  bool
}

func (x *ValueDecl) GetDeclNames() []Name {
	ns := make([]Name, 0, len(x.NameExprs))
	for _, nx := range x.NameExprs {
		if nx.Name == blankIdentifier {
			// ignore
		} else {
			ns = append(ns, nx.Name)
		}
	}
	return ns
}

type TypeDecl struct {
	Attributes
	NameExpr
	Type    Expr // Name, SelectorExpr, StarExpr, or XxxTypes
	IsAlias bool // type alias since Go 1.9
}

func (x *TypeDecl) GetDeclNames() []Name {
	if x.NameExpr.Name == blankIdentifier {
		return nil // ignore
	} else {
		return []Name{x.NameExpr.Name}
	}
}

func HasDeclName(d Decl, n2 Name) bool {
	ns := d.GetDeclNames()
	return slices.Contains(ns, n2)
}

// ----------------------------------------
// SimpleDeclStmt
//
// These are elements of DeclStmt, and get pushed to m.Stmts.

type SimpleDeclStmt interface {
	Decl
	Stmt
	assertSimpleDeclStmt()
}

// not used to avoid itable costs.
// type SimpleDeclStmts []SimpleDeclStmt

func (x *ValueDecl) assertSimpleDeclStmt() {}
func (x *TypeDecl) assertSimpleDeclStmt()  {}

func (x *ValueDecl) assertStmt() {}
func (x *TypeDecl) assertStmt()  {}

var (
	_ SimpleDeclStmt = &ValueDecl{}
	_ SimpleDeclStmt = &TypeDecl{}
)

// ----------------------------------------
// *FileSet

type FileSet struct {
	Files []*FileNode
}

// PackageNameFromFileBody extracts the package name from the given Gno code body.
// The 'name' parameter is used for better error traces, and 'body' contains the Gno code.
func PackageNameFromFileBody(name, body string) (Name, error) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, name, body, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}

	return Name(astFile.Name.Name), nil
}

// MustPackageNameFromFileBody is a wrapper around [PackageNameFromFileBody] that panics on error.
func MustPackageNameFromFileBody(name, body string) Name {
	pkgName, err := PackageNameFromFileBody(name, body)
	if err != nil {
		panic(err)
	}
	return pkgName
}

// ReadMemPackage initializes a new MemPackage by reading the OS directory
// at dir, and saving it with the given pkgPath (import path).
// The resulting MemPackage will contain the names and content of all *.gno files,
// and additionally README.md, LICENSE.
//
// ReadMemPackage does not perform validation aside from the package's name;
// the files are not parsed but their contents are merely stored inside a MemFile.
//
// NOTE: panics if package name is invalid (characters must be alphanumeric or _,
// lowercase, and must start with a letter).
func ReadMemPackage(dir string, pkgPath string) (*gnovm.MemPackage, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	allowedFiles := []string{ // make case insensitive?
		"LICENSE",
		"README.md",
	}
	allowedFileExtensions := []string{
		".gno",
	}
	// exceptions to allowedFileExtensions
	var rejectedFileExtensions []string

	if IsStdlib(pkgPath) {
		// Allows transpilation to work on stdlibs with native fns.
		allowedFileExtensions = append(allowedFileExtensions, ".go")
		rejectedFileExtensions = []string{".gen.go"}
	}

	list := make([]string, 0, len(files))
	for _, file := range files {
		// Ignore directories and hidden files, only include allowed files & extensions,
		// then exclude files that are of the rejected extensions.
		if file.IsDir() ||
			strings.HasPrefix(file.Name(), ".") ||
			(!endsWithAny(file.Name(), allowedFileExtensions) && !slices.Contains(allowedFiles, file.Name())) ||
			endsWithAny(file.Name(), rejectedFileExtensions) {
			continue
		}
		list = append(list, filepath.Join(dir, file.Name()))
	}
	return ReadMemPackageFromList(list, pkgPath)
}

func endsWithAny(str string, suffixes []string) bool {
	return slices.ContainsFunc(suffixes, func(s string) bool {
		return strings.HasSuffix(str, s)
	})
}

// MustReadMemPackage is a wrapper around [ReadMemPackage] that panics on error.
func MustReadMemPackage(dir string, pkgPath string) *gnovm.MemPackage {
	pkg, err := ReadMemPackage(dir, pkgPath)
	if err != nil {
		panic(err)
	}
	return pkg
}

// ReadMemPackageFromList creates a new [gnovm.MemPackage] with the specified pkgPath,
// containing the contents of all the files provided in the list slice.
// No parsing or validation is done on the filenames.
//
// NOTE: errors out if package name is invalid (characters must be alphanumeric or _,
// lowercase, and must start with a letter).
func ReadMemPackageFromList(list []string, pkgPath string) (*gnovm.MemPackage, error) {
	memPkg := &gnovm.MemPackage{Path: pkgPath}
	var pkgName Name
	for _, fpath := range list {
		fname := filepath.Base(fpath)
		bz, err := os.ReadFile(fpath)
		if err != nil {
			return nil, err
		}
		// XXX: should check that all pkg names are the same (else package is invalid)
		if pkgName == "" && strings.HasSuffix(fname, ".gno") {
			pkgName, err = PackageNameFromFileBody(fname, string(bz))
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(string(pkgName), "_test") {
				pkgName = pkgName[:len(pkgName)-len("_test")]
			}
		}
		memPkg.Files = append(memPkg.Files,
			&gnovm.MemFile{
				Name: fname,
				Body: string(bz),
			})
	}

	memPkg.Name = string(pkgName)

	// If no .gno files are present, package simply does not exist.
	if !memPkg.IsEmpty() {
		if err := validatePkgName(string(pkgName)); err != nil {
			return nil, err
		}
	}

	return memPkg, nil
}

// MustReadMemPackageFromList is a wrapper around [ReadMemPackageFromList] that panics on error.
func MustReadMemPackageFromList(list []string, pkgPath string) *gnovm.MemPackage {
	pkg, err := ReadMemPackageFromList(list, pkgPath)
	if err != nil {
		panic(err)
	}
	return pkg
}

// ParseMemPackage executes [ParseFile] on each file of the memPkg, excluding
// test and spurious (non-gno) files. The resulting *FileSet is returned.
//
// If one of the files has a different package name than memPkg.Name,
// or [ParseFile] returns an error, ParseMemPackage panics.
func ParseMemPackage(memPkg *gnovm.MemPackage) (fset *FileSet) {
	fset = &FileSet{}
	var errs error
	for _, mfile := range memPkg.Files {
		if !strings.HasSuffix(mfile.Name, ".gno") ||
			endsWithAny(mfile.Name, []string{"_test.gno", "_filetest.gno"}) {
			continue // skip spurious or test file.
		}
		n, err := ParseFile(mfile.Name, mfile.Body)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		if memPkg.Name != string(n.PkgName) {
			panic(fmt.Sprintf(
				"expected package name [%s] but got [%s]",
				memPkg.Name, n.PkgName))
		}
		// add package file.
		fset.AddFiles(n)
	}
	if errs != nil {
		panic(errs)
	}
	return fset
}

func (fs *FileSet) AddFiles(fns ...*FileNode) {
	fs.Files = append(fs.Files, fns...)
}

func (fs *FileSet) GetFileByName(n Name) *FileNode {
	for _, fn := range fs.Files {
		if fn.Name == n {
			return fn
		}
	}
	return nil
}

// Returns a pointer to the file body decl (as well as
// the *FileNode which contains it) that declares n
// for the associated package with *FileSet.  Does not
// work for import decls which are for the file level.
// The file body decl can be replaced by reference
// assignment.
// TODO move to package?
func (fs *FileSet) GetDeclFor(n Name) (*FileNode, *Decl) {
	fn, decl, ok := fs.GetDeclForSafe(n)
	if !ok {
		panic(fmt.Sprintf(
			"name %s not defined in fileset with files %v",
			n, fs.FileNames()))
	}
	return fn, decl
}

func (fs *FileSet) GetDeclForSafe(n Name) (*FileNode, *Decl, bool) {
	// XXX index to bound to linear time.

	// Iteration happens reversing fs.Files; this is because the LAST declaration
	// of n is what we are looking for.
	for i := len(fs.Files) - 1; i >= 0; i-- {
		fn := fs.Files[i]
		for i, dn := range fn.Decls {
			if _, isImport := dn.(*ImportDecl); isImport {
				// imports in other files don't count.
				continue
			}
			if HasDeclName(dn, n) {
				// found the decl that declares n.
				return fn, &fn.Decls[i], true
			}
		}
	}
	return nil, nil, false
}

func (fs *FileSet) FileNames() []string {
	res := make([]string, len(fs.Files))
	for i, fn := range fs.Files {
		res[i] = string(fn.Name)
	}
	return res
}

// ----------------------------------------
// FileNode, & PackageNode

type FileNode struct {
	Attributes
	StaticBlock
	Name
	PkgName Name
	Decls
}

type PackageNode struct {
	Attributes
	StaticBlock
	PkgPath string
	PkgName Name
	*FileSet
}

func PackageNodeLocation(path string) Location {
	return Location{
		PkgPath: path,
		File:    "",
		Line:    0,
	}
}

func NewPackageNode(name Name, path string, fset *FileSet) *PackageNode {
	pn := &PackageNode{
		PkgPath: path,
		PkgName: name,
		FileSet: fset,
	}
	pn.SetLocation(PackageNodeLocation(path))
	pn.InitStaticBlock(pn, nil)
	return pn
}

func (x *PackageNode) NewPackage() *PackageValue {
	pv := &PackageValue{
		Block: &Block{
			Source: x,
		},
		PkgName:    x.PkgName,
		PkgPath:    x.PkgPath,
		FNames:     nil,
		FBlocks:    nil,
		fBlocksMap: make(map[Name]*Block),
	}
	if IsRealmPath(x.PkgPath) || x.PkgPath == "main" {
		rlm := NewRealm(x.PkgPath)
		pv.SetRealm(rlm)
	}
	pv.IncRefCount() // all package values have starting ref count of 1.
	x.PrepareNewValues(pv)
	return pv
}

// Prepares new func values (e.g. by attaching the proper file block closure).
// Returns a slice of new PackageValue.Values.
// After return, *PackageNode.Values and *PackageValue.Values have the same
// length.
// NOTE: declared methods do not get their closures set here. See
// *DeclaredType.GetValueAt() which returns a filled copy.
func (x *PackageNode) PrepareNewValues(pv *PackageValue) []TypedValue {
	// should already exist.
	block := pv.Block.(*Block)
	if block.Source != x {
		// special case if block.Source is ref node
		if ref, ok := block.Source.(RefNode); ok && ref.Location == PackageNodeLocation(pv.PkgPath) {
			// this is fine
		} else {
			panic("PackageNode.PrepareNewValues() package mismatch")
		}
	}
	// The FuncValue Body may have been altered during the preprocessing.
	// We need to update body field from the source in the FuncValue accordingly.
	for _, tv := range x.Values {
		if fv, ok := tv.V.(*FuncValue); ok {
			fv.UpdateBodyFromSource()
		}
	}
	pvl := len(block.Values)
	pnl := len(x.Values)
	// copy new top-level defined values/types.
	if pvl < pnl {
		nvs := make([]TypedValue, pnl-pvl)
		copy(nvs, x.Values[pvl:pnl])
		for i, tv := range nvs {
			if fv, ok := tv.V.(*FuncValue); ok {
				// copy function value and assign closure from package value.
				fv = fv.Copy(nilAllocator)
				fv.Parent = pv.fBlocksMap[fv.FileName]
				if fv.Parent == nil {
					panic(fmt.Sprintf("file block missing for file %q", fv.FileName))
				}
				nvs[i].V = fv
			}
		}
		heapItems := x.GetHeapItems()
		for i, tv := range nvs {
			if _, ok := tv.T.(heapItemType); ok {
				panic("unexpected heap item")
			}
			if heapItems[pvl+i] {
				nvs[i] = TypedValue{
					T: heapItemType{},
					V: &HeapItemValue{Value: nvs[i]},
				}
			}
		}
		block.Values = append(block.Values, nvs...)
		return block.Values[pvl:]
	} else if pvl > pnl {
		panic("package size error")
	} else {
		// nothing to do
		return nil
	}
}

// DefineNativeFunc defines a native function.
func (x *PackageNode) DefineNative(n Name, ps, rs FieldTypeExprs, native func(*Machine)) {
	if debug {
		debug.Printf("*PackageNode.DefineNative(%s,...)\n", n)
	}
	if native == nil {
		panic("DefineNative expects a function, but got nil")
	}

	fd := FuncD(n, ps, rs, nil)
	fd = Preprocess(nil, x, fd).(*FuncDecl)
	ft := evalStaticType(nil, x, &fd.Type).(*FuncType)
	if debug {
		if ft == nil {
			panic("should not happen")
		}
	}
	fv := x.GetValueRef(nil, n, true).V.(*FuncValue)
	fv.nativeBody = native
}

// Same as DefineNative but allow the overriding of previously defined natives.
// For example, overriding a native function defined in stdlibs/stdlibs for
// testing. Caller must ensure that the function type is identical.
func (x *PackageNode) DefineNativeOverride(n Name, native func(*Machine)) {
	if debug {
		debug.Printf("*PackageNode.DefineNativeOverride(%s,...)\n", n)
	}
	if native == nil {
		panic("DefineNative expects a function, but got nil")
	}
	fv := x.GetValueRef(nil, n, true).V.(*FuncValue)
	fv.nativeBody = native
}

// ----------------------------------------
// RefNode

// Reference to a node by its location.
type RefNode struct {
	Location  Location // location of node.
	BlockNode          // convenience to implement BlockNode (nil).
}

func (rn RefNode) GetLocation() Location {
	return rn.Location
}

// ----------------------------------------
// BlockNode

// Nodes that create their own scope satisfy this interface.
type BlockNode interface {
	Node
	InitStaticBlock(BlockNode, BlockNode)
	IsInitialized() bool
	GetStaticBlock() *StaticBlock
	GetLocation() Location
	SetLocation(Location)

	// StaticBlock promoted methods
	GetBlockNames() []Name
	GetExternNames() []Name
	GetNumNames() uint16
	SetIsHeapItem(n Name)
	GetHeapItems() []bool
	GetParentNode(Store) BlockNode
	GetPathForName(Store, Name) ValuePath
	GetBlockNodeForPath(Store, ValuePath) BlockNode
	GetIsConst(Store, Name) bool
	GetIsConstAt(Store, ValuePath) bool
	GetLocalIndex(Name) (uint16, bool)
	GetValueRef(Store, Name, bool) *TypedValue
	GetStaticTypeOf(Store, Name) Type
	GetStaticTypeOfAt(Store, ValuePath) Type
	Predefine(bool, Name)
	Define(Name, TypedValue)
	Define2(bool, Name, Type, TypedValue)
	GetBody() Body
	SetBody(Body)
}

// ----------------------------------------
// StaticBlock

// Embed in node to make it a BlockNode.
type StaticBlock struct {
	Block
	Types             []Type
	NumNames          uint16
	Names             []Name
	HeapItems         []bool
	UnassignableNames []Name
	Consts            []Name // TODO consider merging with Names.
	Externs           []Name
	Parent            BlockNode
	Loc               Location

	// temporary storage for rolling back redefinitions.
	oldValues []oldValue
}

type oldValue struct {
	idx   uint16
	value Value
}

// revert values upon failure of redefinitions.
func (sb *StaticBlock) revertToOld() {
	for _, ov := range sb.oldValues {
		sb.Block.Values[ov.idx].V = ov.value
	}
	sb.oldValues = nil
}

// Implements BlockNode
func (sb *StaticBlock) InitStaticBlock(source BlockNode, parent BlockNode) {
	if sb.Names != nil || sb.Block.Source != nil {
		panic("StaticBlock already initialized")
	}
	if parent == nil {
		sb.Block = Block{
			Source: source,
			Values: nil,
			Parent: nil,
		}
	} else {
		switch source.(type) {
		case *IfCaseStmt, *SwitchClauseStmt:
			if parent == nil {
				sb.Block = Block{
					Source: source,
					Values: nil,
					Parent: nil,
				}
			} else {
				parent2 := parent.GetParentNode(nil)
				sb.Block = Block{
					Source: source,
					Values: nil,
					Parent: parent2.GetStaticBlock().GetBlock(),
				}
			}
		default:
			sb.Block = Block{
				Source: source,
				Values: nil,
				Parent: parent.GetStaticBlock().GetBlock(),
			}
		}
	}
	sb.NumNames = 0
	sb.Names = make([]Name, 0, 16)
	sb.HeapItems = make([]bool, 0, 16)
	sb.Consts = make([]Name, 0, 16)
	sb.Externs = make([]Name, 0, 16)
	sb.Parent = parent
	return
}

// Implements BlockNode.
func (sb *StaticBlock) IsInitialized() bool {
	return sb.Block.Source != nil
}

// Implements BlockNode.
func (sb *StaticBlock) GetStaticBlock() *StaticBlock {
	return sb
}

// Implements BlockNode.
func (sb *StaticBlock) GetLocation() Location {
	return sb.Loc
}

// Implements BlockNode.
func (sb *StaticBlock) SetLocation(loc Location) {
	sb.Loc = loc
}

// Does not implement BlockNode to prevent confusion.
// To get the static *Block, call Blocknode.GetStaticBlock().GetBlock().
func (sb *StaticBlock) GetBlock() *Block {
	return &sb.Block
}

// Implements BlockNode.
func (sb *StaticBlock) GetBlockNames() (ns []Name) {
	return sb.Names
}

// Implements BlockNode.
// NOTE: Extern names may also be local, if declared after usage as an extern
// (thus shadowing the extern name).
func (sb *StaticBlock) GetExternNames() (ns []Name) {
	return sb.Externs
}

func (sb *StaticBlock) addExternName(n Name) {
	if slices.Contains(sb.Externs, n) {
		return
	}
	sb.Externs = append(sb.Externs, n)
}

// Implements BlockNode.
func (sb *StaticBlock) GetNumNames() (nn uint16) {
	return sb.NumNames
}

// Implements BlockNode.
func (sb *StaticBlock) GetHeapItems() []bool {
	return sb.HeapItems
}

// Implements BlockNode.
func (sb *StaticBlock) SetIsHeapItem(n Name) {
	idx, ok := sb.GetLocalIndex(n)
	if !ok {
		panic("name not found in block")
	}
	sb.HeapItems[idx] = true
}

// Implements BlockNode.
func (sb *StaticBlock) GetParentNode(store Store) BlockNode {
	return sb.Parent
}

// Implements BlockNode.
// As a side effect, notes externally defined names.
// Slow, for precompile only.
func (sb *StaticBlock) GetPathForName(store Store, n Name) ValuePath {
	if n == blankIdentifier {
		return NewValuePathBlock(0, 0, blankIdentifier)
	}
	// Check local.
	gen := 1
	if idx, ok := sb.GetLocalIndex(n); ok {
		return NewValuePathBlock(uint8(gen), idx, n)
	}
	sn := sb.GetSource(store)
	// Register as extern.
	// NOTE: uverse names are externs too.
	// NOTE: externs may also be shadowed later in the block. Thus, usages
	// before the declaration will have depth > 1; following it, depth == 1,
	// matching the two different identifiers they refer to.
	if !isFile(sn) {
		sb.GetStaticBlock().addExternName(n)
	}
	// Check ancestors.
	gen++
	fauxChild := 0
	if fauxChildBlockNode(sn) {
		fauxChild++
	}
	sn = sn.GetParentNode(store)
	for sn != nil {
		if idx, ok := sn.GetLocalIndex(n); ok {
			if 0xff < (gen - fauxChild) {
				panic("value path depth overflow")
			}
			return NewValuePathBlock(uint8(gen-fauxChild), idx, n)
		} else {
			if !isFile(sn) {
				sn.GetStaticBlock().addExternName(n)
			}
			gen++
			if fauxChildBlockNode(sn) {
				fauxChild++
			}
			sn = sn.GetParentNode(store)
		}
	}
	// Finally, check uverse.
	if idx, ok := UverseNode().GetLocalIndex(n); ok {
		return NewValuePathUverse(idx, n)
	}
	// Name does not exist.
	panic(fmt.Sprintf("name %s not declared", n))
}

// Get the containing block node for node with path relative to this containing block.
// Slow, for precompile only.
func (sb *StaticBlock) GetBlockNodeForPath(store Store, path ValuePath) BlockNode {
	if path.Type != VPBlock {
		panic("expected block type value path but got " + path.Type.String())
	}

	// NOTE: path.Depth == 1 means it's in bn.
	bn := sb.GetSource(store)
	for i := 1; i < int(path.Depth); i++ {
		if fauxChildBlockNode(bn) {
			bn = bn.GetParentNode(store)
		}
		bn = bn.GetParentNode(store)
	}

	// If bn is a faux child block node, check also its faux parent.
	switch bn := bn.(type) {
	case *IfCaseStmt, *SwitchClauseStmt:
		pn := bn.GetParentNode(store)
		if path.Index < pn.GetNumNames() {
			return pn
		}
	}

	return bn
}

// Returns whether a name defined here in in ancestry is a const.
// This is not the same as whether a name's static type is
// untyped -- as in c := a == b, a name may be an untyped non-const.
// Implements BlockNode.
func (sb *StaticBlock) GetIsConst(store Store, n Name) bool {
	_, ok := sb.GetLocalIndex(n)
	bp := sb.GetParentNode(store)
	for {
		if ok {
			return sb.getLocalIsConst(n)
		} else if bp != nil {
			_, ok = bp.GetLocalIndex(n)
			sb = bp.GetStaticBlock()
			bp = bp.GetParentNode(store)
		} else {
			panic(fmt.Sprintf("name %s not declared", n))
		}
	}
}

func (sb *StaticBlock) GetIsConstAt(store Store, path ValuePath) bool {
	return sb.GetBlockNodeForPath(store, path).GetStaticBlock().getLocalIsConst(path.Name)
}

// Returns true iff n is a local const defined name.
func (sb *StaticBlock) getLocalIsConst(n Name) bool {
	return slices.Contains(sb.Consts, n)
}

func (sb *StaticBlock) IsAssignable(store Store, n Name) bool {
	_, ok := sb.GetLocalIndex(n)
	bp := sb.GetParentNode(store)
	un := sb.UnassignableNames

	for {
		if ok {
			return !slices.Contains(un, n)
		} else if bp != nil {
			_, ok = bp.GetLocalIndex(n)
			un = bp.GetStaticBlock().UnassignableNames
			bp = bp.GetParentNode(store)
		} else if _, ok := UverseNode().GetLocalIndex(n); ok {
			return false
		} else {
			return true
		}
	}
}

// Implements BlockNode.
func (sb *StaticBlock) GetStaticTypeOf(store Store, n Name) Type {
	idx, ok := sb.GetLocalIndex(n)
	ts := sb.Types
	bp := sb.GetParentNode(store)
	for {
		if ok {
			return ts[idx]
		} else if bp != nil {
			idx, ok = bp.GetLocalIndex(n)
			ts = bp.GetStaticBlock().Types
			bp = bp.GetParentNode(store)
		} else if idx, ok := UverseNode().GetLocalIndex(n); ok {
			path := NewValuePathUverse(idx, n)
			tv := Uverse().GetValueAt(store, path)
			return tv.T
		} else {
			panic(fmt.Sprintf("name %s not declared", n))
		}
	}
}

// Implements BlockNode.
func (sb *StaticBlock) GetStaticTypeOfAt(store Store, path ValuePath) Type {
	if debug {
		if path.Depth == 0 {
			panic("should not happen")
		}
	}
	bn := sb.GetBlockNodeForPath(store, path)
	return bn.GetStaticBlock().Types[path.Index]
}

// Implements BlockNode.
func (sb *StaticBlock) GetLocalIndex(n Name) (uint16, bool) {
	for i, name := range sb.Names {
		if name == n {
			if debug {
				nt := reflect.TypeOf(sb.Source).String()
				debug.Printf("StaticBlock(%p %v).GetLocalIndex(%s) = %v, %v\n",
					sb, nt, n, i, name)
			}
			return uint16(i), true
		}
	}
	if debug {
		nt := reflect.TypeOf(sb.Source).String()
		debug.Printf("StaticBlock(%p %v).GetLocalIndex(%s) = undefined\n",
			sb, nt, n)
	}
	return 0, false
}

// Implemented BlockNode.
// This method is too slow for runtime, but it is used
// during preprocessing to compute types.
// If skipPredefined, skips over names that are only predefined.
// Returns nil if not defined.
func (sb *StaticBlock) GetValueRef(store Store, n Name, skipPredefined bool) *TypedValue {
	idx, ok := sb.GetLocalIndex(n)
	bb := &sb.Block
	bp := sb.GetParentNode(store)
	for {
		if ok && (!skipPredefined || sb.Types[idx] != nil) {
			return bb.GetPointerToInt(store, int(idx)).TV
		} else if bp != nil {
			idx, ok = bp.GetLocalIndex(n)
			sb = bp.GetStaticBlock()
			bb = sb.GetBlock()
			bp = bp.GetParentNode(store)
		} else {
			return nil
		}
	}
}

// Implements BlockNode
// Statically declares a name definition.
// At runtime, use *Block.GetPointerTo() which takes a path
// value, which is pre-computeed in the preprocessor.
// Once a typed value is defined, it cannot be changed.
//
// NOTE: Currently tv.V is only set when the value represents a Type(Value) or
// a FuncValue.  The purpose of tv is to describe the invariant of a named
// value, at the minimum its type, but also sometimes the typeval value; but we
// could go further and store preprocessed constant results here too.  See
// "anyValue()" and "asValue()" for usage.
func (sb *StaticBlock) Define(n Name, tv TypedValue) {
	sb.Define2(false, n, tv.T, tv)
}

// Set type to nil, only reserving the name.
func (sb *StaticBlock) Predefine(isConst bool, n Name) {
	_, exists := sb.GetLocalIndex(n)
	if !exists {
		sb.Define2(isConst, n, nil, anyValue(nil))
	}
}

// The declared type st may not be the same as the static tv;
// e.g. var x MyInterface = MyStruct{}.
// Setting st and tv to nil/zero reserves (predefines) name for definition later.
func (sb *StaticBlock) Define2(isConst bool, n Name, st Type, tv TypedValue) {
	if debug {
		debug.Printf(
			"StaticBlock.Define2(%v, %s, %v, %v)\n",
			isConst, n, st, tv)
	}
	// TODO check that tv.T implements t.
	if len(n) == 0 {
		panic("name cannot be zero")
	}
	if int(sb.NumNames) != len(sb.Names) {
		panic("StaticBlock.NumNames and len(.Names) mismatch")
	}
	if sb.NumNames == math.MaxUint16 {
		panic("too many variables in block")
	}
	if tv.T == nil && tv.V != nil {
		panic("StaticBlock.Define2() requires .T if .V is set")
	}
	if n == blankIdentifier {
		return // ignore
	}
	idx, exists := sb.GetLocalIndex(n)
	if exists {
		// Is re-defining.
		if isConst != sb.getLocalIsConst(n) {
			panic(fmt.Sprintf(
				"StaticBlock.Define2(%s) cannot change const status",
				n))
		}
		old := sb.Block.Values[idx]
		if !old.IsUndefined() && tv.T != nil {
			if tv.T.Kind() == FuncKind && baseOf(tv.T).(*FuncType).IsZero() {
				// special case,
				// allow re-predefining for func upgrades.
				// keep the old type so we can check it at preprocessor.
				tv.T = old.T
				fv := tv.V.(*FuncValue)
				fv.Type = old.T
				st = old.T
				sb.oldValues = append(sb.oldValues,
					oldValue{idx, old.V})
			} else {
				if tv.T.TypeID() != old.T.TypeID() {
					panic(fmt.Sprintf(
						"StaticBlock.Define2(%s) cannot change .T; was %v, new %v",
						n, old.T, tv.T))
				}
				if tv.V != old.V {
					panic(fmt.Sprintf(
						"StaticBlock.Define2(%s) cannot change .V",
						n))
				}
			}
			// Allow re-definitions if they have the same type.
			// (In normal scenarios, duplicate declarations are "caught" by RunMemPackage.)
		}
		sb.Block.Values[idx] = tv
		sb.Types[idx] = st
	} else {
		// The general case without re-definition.
		sb.Names = append(sb.Names, n)
		sb.HeapItems = append(sb.HeapItems, false)
		if isConst {
			sb.Consts = append(sb.Consts, n)
		}
		sb.NumNames++
		sb.Block.Values = append(sb.Block.Values, tv)
		sb.Types = append(sb.Types, st)
	}
}

// Implements BlockNode
func (sb *StaticBlock) SetStaticBlock(osb StaticBlock) {
	*sb = osb
}

var (
	_ BlockNode = &FuncLitExpr{}
	_ BlockNode = &BlockStmt{}
	_ BlockNode = &ForStmt{}
	_ BlockNode = &IfStmt{} // faux block node
	_ BlockNode = &IfCaseStmt{}
	_ BlockNode = &RangeStmt{}
	_ BlockNode = &SelectCaseStmt{}
	_ BlockNode = &SwitchStmt{} // faux block node
	_ BlockNode = &SwitchClauseStmt{}
	_ BlockNode = &FuncDecl{}
	_ BlockNode = &FileNode{}
	_ BlockNode = &PackageNode{}
	_ BlockNode = RefNode{}
)

func (x *IfStmt) GetBody() Body {
	panic("IfStmt has no body (but .Then and .Else do)")
}

func (x *IfStmt) SetBody(b Body) {
	panic("IfStmt has no body (but .Then and .Else do)")
}

func (x *SwitchStmt) GetBody() Body {
	panic("SwitchStmt has no body (but its cases do)")
}

func (x *SwitchStmt) SetBody(b Body) {
	panic("SwitchStmt has no body (but its cases do)")
}

func (x *FileNode) GetBody() Body {
	panic("FileNode has no body (but it does have .Decls)")
}

func (x *FileNode) SetBody(b Body) {
	panic("FileNode has no body (but it does have .Decls)")
}

func (x *PackageNode) GetBody() Body {
	panic("PackageNode has no body")
}

func (x *PackageNode) SetBody(b Body) {
	panic("PackageNode has no body")
}

// ----------------------------------------
// Value Path

// A relative pointer to a TypedValue value
//
//	(a) a Block scope var or const
//	(b) a StructValue field
//	(c) a DeclaredType method
//	(d) a PackageNode declaration
//
// Depth tells how many layers of access should be unvealed before
// arriving at the ultimate handler type.  In the case of Blocks,
// the depth tells how many layers of ancestry to ascend before
// arriving at the target block.  For other selector expr paths
// such as those for *DeclaredType methods or *StructType fields,
// see tests/selector_test.go.
type ValuePath struct {
	Type VPType // see VPType* consts.
	// Warning: Use SetDepth() to set Depth.
	Depth uint8  // see doc for ValuePath.
	Index uint16 // index of value, field, or method.
	Name  Name   // name of value, field, or method.
}

// Maximum depth of a ValuePath.
const MaxValuePathDepth = 127

func (vp ValuePath) validateDepth() {
	if vp.Depth > MaxValuePathDepth {
		panic(fmt.Sprintf("exceeded maximum %s depth (%d)", vp.Type, MaxValuePathDepth))
	}
}

func (vp *ValuePath) SetDepth(d uint8) {
	vp.Depth = d

	vp.validateDepth()
}

type VPType uint8

const (
	VPUverse         VPType = 0x00
	VPBlock          VPType = 0x01 // blocks and packages
	VPField          VPType = 0x02
	VPValMethod      VPType = 0x03
	VPPtrMethod      VPType = 0x04
	VPInterface      VPType = 0x05
	VPSubrefField    VPType = 0x06 // not deref type
	VPDerefField     VPType = 0x12 // 0x10 + VPField
	VPDerefValMethod VPType = 0x13 // 0x10 + VPValMethod
	VPDerefPtrMethod VPType = 0x14 // 0x10 + VPPtrMethod
	VPDerefInterface VPType = 0x15 // 0x10 + VPInterface
	// 0x3X, 0x5X, 0x7X, 0x9X, 0xAX, 0xCX, 0xEX reserved.
)

func NewValuePath(t VPType, depth uint8, index uint16, n Name) ValuePath {
	vp := ValuePath{
		Type:  t,
		Depth: depth,
		Index: index,
		Name:  n,
	}
	vp.Validate()
	return vp
}

func NewValuePathUverse(index uint16, n Name) ValuePath {
	return NewValuePath(VPUverse, 0, index, n)
}

func NewValuePathBlock(depth uint8, index uint16, n Name) ValuePath {
	return NewValuePath(VPBlock, depth, index, n)
}

func NewValuePathField(depth uint8, index uint16, n Name) ValuePath {
	return NewValuePath(VPField, depth, index, n)
}

func NewValuePathValMethod(index uint16, n Name) ValuePath {
	return NewValuePath(VPValMethod, 0, index, n)
}

func NewValuePathPtrMethod(index uint16, n Name) ValuePath {
	return NewValuePath(VPPtrMethod, 0, index, n)
}

func NewValuePathInterface(n Name) ValuePath {
	return NewValuePath(VPInterface, 0, 0, n)
}

func NewValuePathSubrefField(depth uint8, index uint16, n Name) ValuePath {
	return NewValuePath(VPSubrefField, depth, index, n)
}

func NewValuePathDerefField(depth uint8, index uint16, n Name) ValuePath {
	return NewValuePath(VPDerefField, depth, index, n)
}

func NewValuePathDerefValMethod(index uint16, n Name) ValuePath {
	return NewValuePath(VPDerefValMethod, 0, index, n)
}

func NewValuePathDerefPtrMethod(index uint16, n Name) ValuePath {
	return NewValuePath(VPDerefPtrMethod, 0, index, n)
}

func NewValuePathDerefInterface(n Name) ValuePath {
	return NewValuePath(VPDerefInterface, 0, 0, n)
}

func (vp ValuePath) Validate() {
	vp.validateDepth()

	switch vp.Type {
	case VPUverse:
		if vp.Depth != 0 {
			panic("uverse value path must have depth 0")
		}
	case VPBlock:
		// 0 ok ("_" blank)
	case VPField:
		if vp.Depth > 1 {
			panic("field value path must have depth 0 or 1")
		}
	case VPValMethod:
		if vp.Depth != 0 {
			panic("method value path must have depth 0")
		}
	case VPPtrMethod:
		if vp.Depth != 0 {
			panic("ptr receiver method value path must have depth 0")
		}
	case VPInterface:
		if vp.Depth != 0 {
			panic("interface method value path must have depth 0")
		}
		if vp.Name == "" {
			panic("interface value path must have name")
		}
	case VPSubrefField:
		if vp.Depth > 3 {
			panic("subref field value path must have depth 0, 1, 2, or 3")
		}
	case VPDerefField:
		if vp.Depth > 3 {
			panic("deref field value path must have depth 0, 1, 2, or 3")
		}
	case VPDerefValMethod:
		if vp.Depth != 0 {
			panic("(deref) method value path must have depth 0")
		}
	case VPDerefPtrMethod:
		if vp.Depth != 0 {
			panic("(deref) ptr receiver method value path must have depth 0")
		}
	case VPDerefInterface:
		if vp.Depth != 0 {
			panic("(deref) interface method value path must have depth 0")
		}
		if vp.Name == "" {
			panic("(deref) interface value path must have name")
		}
	default:
		panic(fmt.Sprintf(
			"unexpected value path type %X",
			vp.Type))
	}
}

func (vp ValuePath) IsBlockBlankPath() bool {
	return vp.Type == VPBlock && vp.Depth == 0 && vp.Index == 0
}

func (vp ValuePath) IsDerefType() bool {
	return vp.Type&0x10 > 0
}

type ValuePather interface {
	GetPathForName(Name) ValuePath
}

// ----------------------------------------
// Utility

func (x *BasicLitExpr) GetString() string {
	// Matches string literal parsing in go/constant.MakeFromLiteral.
	str, err := strconv.Unquote(x.Value)
	if err != nil {
		panic("error in parsing string literal: " + err.Error())
	}
	return str
}

func (x *BasicLitExpr) GetInt() int {
	i, err := strconv.Atoi(x.Value)
	if err != nil {
		panic(err)
	}
	return i
}

var rePkgName = regexp.MustCompile(`^[a-z][a-z0-9_]+$`)

// TODO: consider length restrictions.
// If this function is changed, ReadMemPackage's documentation should be updated accordingly.
func validatePkgName(name string) error {
	if !rePkgName.MatchString(name) {
		return fmt.Errorf("cannot create package with invalid name %q", name)
	}
	return nil
}

// The distinction is used for validation to work
// both before and after preprocessing.
const (
	missingResultNamePrefix    = ".res." // if there was no name
	underscoreResultNamePrefix = ".res_" // if was underscore
)

//nolint:unused
func isUnnamedResult(name Name) bool {
	return isMissingResult(name) || isUnderscoreResult(name)
}

func isMissingResult(name Name) bool {
	return strings.HasPrefix(string(name), missingResultNamePrefix)
}

//nolint:unused
func isUnderscoreResult(name Name) bool {
	return strings.HasPrefix(string(name), underscoreResultNamePrefix)
}
