package parser

import (
	"strings"
)

type OrderDirection string

const (
	OrderDirectionNone OrderDirection = "None"
	OrderDirectionAsc  OrderDirection = "ASC"
	OrderDirectionDesc OrderDirection = "DESC"
)

type Expr interface {
	Pos() Pos
	End() Pos
	String(level int) string
	Accept(visitor ASTVisitor) error
}

type DDL interface {
	Expr
	Type() string
}

type OperationExpr struct {
	OperationPos Pos
	Kind         TokenKind
}

func (o *OperationExpr) Pos() Pos {
	return o.OperationPos
}

func (o *OperationExpr) End() Pos {
	return o.OperationPos + Pos(len(o.Kind))
}

func (o *OperationExpr) String(int) string {
	return strings.ToUpper(string(o.Kind))
}

func (o *OperationExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	return visitor.VisitOperationExpr(o)
}

type TernaryExpr struct {
	Condition Expr
	TrueExpr  Expr
	FalseExpr Expr
}

func (t *TernaryExpr) Pos() Pos {
	return t.Condition.Pos()
}

func (t *TernaryExpr) End() Pos {
	return t.FalseExpr.End()
}

func (t *TernaryExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(t.Condition.String(level))
	builder.WriteString(" ? ")
	builder.WriteString(t.TrueExpr.String(level))
	builder.WriteString(" : ")
	builder.WriteString(t.FalseExpr.String(level))
	return builder.String()
}

func (t *TernaryExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.TrueExpr.Accept(visitor); err != nil {
		return err
	}
	if err := t.FalseExpr.Accept(visitor); err != nil {
		return err
	}
	if err := t.Condition.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTernaryExpr(t)
}

type BinaryExpr struct {
	LeftExpr  Expr
	Operation TokenKind
	RightExpr Expr
	HasGlobal bool
	HasNot    bool
}

func (p *BinaryExpr) Pos() Pos {
	return p.LeftExpr.Pos()
}

func (p *BinaryExpr) End() Pos {
	return p.RightExpr.End()
}

func (p *BinaryExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(p.LeftExpr.String(level))
	if p.Operation != opTypeCast {
		builder.WriteByte(' ')
	}
	if p.HasNot {
		builder.WriteString("NOT ")
	} else if p.HasGlobal {
		builder.WriteString("GLOBAL ")
	}
	builder.WriteString(string(p.Operation))
	if p.Operation != opTypeCast {
		builder.WriteByte(' ')
	}
	builder.WriteString(p.RightExpr.String(level))
	return builder.String()
}

func (p *BinaryExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(p)
	defer visitor.leave(p)
	if err := p.LeftExpr.Accept(visitor); err != nil {
		return err
	}
	if err := p.RightExpr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitBinaryExpr(p)
}

type JoinTableExpr struct {
	Table        *TableExpr
	StatementEnd Pos
	SampleRatio  *SampleRatioExpr
	HasFinal     bool
}

func (j *JoinTableExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(j)
	defer visitor.leave(j)
	if err := j.Table.Accept(visitor); err != nil {
		return err
	}
	if j.SampleRatio != nil {
		return j.SampleRatio.Accept(visitor)
	}
	return visitor.VisitJoinTableExpr(j)
}

func (j *JoinTableExpr) Pos() Pos {
	return j.Table.Pos()
}

func (j *JoinTableExpr) End() Pos {
	return j.StatementEnd
}

func (j *JoinTableExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(j.Table.String(level))
	if j.SampleRatio != nil {
		builder.WriteByte(' ')
		builder.WriteString(j.SampleRatio.String(level))
	}
	if j.HasFinal {
		builder.WriteString(" FINAL")
	}
	return builder.String()
}

type AlterTableExpr interface {
	Expr
	AlterType() string
}

type AlterTable struct {
	AlterPos        Pos
	StatementEnd    Pos
	TableIdentifier *TableIdentifier
	OnCluster       *OnClusterExpr
	AlterExprs      []AlterTableExpr
}

func (a *AlterTable) Pos() Pos {
	return a.AlterPos
}

func (a *AlterTable) End() Pos {
	return a.StatementEnd
}

func (a *AlterTable) Type() string {
	return "ALTER TABLE"
}

func (a *AlterTable) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ALTER TABLE ")
	builder.WriteString(a.TableIdentifier.String(level))
	if a.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(a.OnCluster.String(level))
	}
	for i, expr := range a.AlterExprs {
		builder.WriteString(NewLine(level))
		builder.WriteString(expr.String(level))
		if i != len(a.AlterExprs)-1 {
			builder.WriteString(",")
		}
	}
	return builder.String()
}

func (a *AlterTable) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if a.OnCluster != nil {
		if err := a.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}

	for _, expr := range a.AlterExprs {
		if err := expr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTable(a)
}

type AlterTableAttachPartition struct {
	AttachPos Pos

	Partition *PartitionExpr
	From      *TableIdentifier
}

func (a *AlterTableAttachPartition) Pos() Pos {
	return a.AttachPos
}

func (a *AlterTableAttachPartition) End() Pos {
	if a.From != nil {
		return a.From.End()
	}
	return a.Partition.End()
}

func (a *AlterTableAttachPartition) AlterType() string {
	return "ATTACH_PARTITION"
}

func (a *AlterTableAttachPartition) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ATTACH ")
	builder.WriteString(a.Partition.String(level))
	if a.From != nil {
		builder.WriteString(" FROM ")
		builder.WriteString(a.From.String(level))
	}
	return builder.String()
}

func (a *AlterTableAttachPartition) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Partition.Accept(visitor); err != nil {
		return err
	}
	if a.From != nil {
		if err := a.From.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableAttachPartition(a)
}

type AlterTableDetachPartition struct {
	DetachPos Pos
	Partition *PartitionExpr
	Settings  *SettingsExprList
}

func (a *AlterTableDetachPartition) Pos() Pos {
	return a.DetachPos
}

func (a *AlterTableDetachPartition) End() Pos {
	return a.Partition.End()
}

func (a *AlterTableDetachPartition) AlterType() string {
	return "DETACH_PARTITION"
}

func (a *AlterTableDetachPartition) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DETACH ")
	builder.WriteString(a.Partition.String(level))
	if a.Settings != nil {
		builder.WriteByte(' ')
		builder.WriteString(a.Settings.String(level))
	}
	return builder.String()
}

func (a *AlterTableDetachPartition) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Partition.Accept(visitor); err != nil {
		return err
	}
	if a.Settings != nil {
		if err := a.Settings.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableDetachPartition(a)
}

type AlterTableDropPartition struct {
	DropPos   Pos
	Partition *PartitionExpr
}

func (a *AlterTableDropPartition) Pos() Pos {
	return a.DropPos
}

func (a *AlterTableDropPartition) End() Pos {
	return a.Partition.End()
}

func (a *AlterTableDropPartition) AlterType() string {
	return "DROP_PARTITION"
}

func (a *AlterTableDropPartition) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP ")
	builder.WriteString(a.Partition.String(level))
	return builder.String()
}

func (a *AlterTableDropPartition) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Partition.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableDropPartition(a)
}

type AlterTableFreezePartition struct {
	FreezePos    Pos
	StatementEnd Pos
	Partition    *PartitionExpr
}

func (a *AlterTableFreezePartition) Pos() Pos {
	return a.FreezePos
}

func (a *AlterTableFreezePartition) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableFreezePartition) AlterType() string {
	return "FREEZE_PARTITION"
}

func (a *AlterTableFreezePartition) String(level int) string {
	var builder strings.Builder
	builder.WriteString("FREEZE")
	if a.Partition != nil {
		builder.WriteByte(' ')
		builder.WriteString(a.Partition.String(level))
	}
	return builder.String()
}

func (a *AlterTableFreezePartition) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if a.Partition != nil {
		if err := a.Partition.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableFreezePartition(a)
}

type AlterTableAddColumn struct {
	AddPos       Pos
	StatementEnd Pos

	Column      *Column
	IfNotExists bool
	After       *NestedIdentifier
}

func (a *AlterTableAddColumn) Pos() Pos {
	return a.AddPos
}

func (a *AlterTableAddColumn) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableAddColumn) AlterType() string {
	return "ADD_COLUMN"
}

func (a *AlterTableAddColumn) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ADD COLUMN ")
	builder.WriteString(a.Column.String(level))
	if a.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	if a.After != nil {
		builder.WriteString(" AFTER ")
		builder.WriteString(a.After.String(level))
	}
	return builder.String()
}

func (a *AlterTableAddColumn) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Column.Accept(visitor); err != nil {
		return err
	}
	if a.After != nil {
		if err := a.After.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableAddColumn(a)
}

type AlterTableAddIndex struct {
	AddPos       Pos
	StatementEnd Pos

	Index       *TableIndex
	IfNotExists bool
	After       *NestedIdentifier
}

func (a *AlterTableAddIndex) Pos() Pos {
	return a.AddPos
}

func (a *AlterTableAddIndex) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableAddIndex) AlterType() string {
	return "ADD_INDEX"
}

func (a *AlterTableAddIndex) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ADD INDEX ")
	builder.WriteString(a.Index.String(level))
	if a.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	if a.After != nil {
		builder.WriteString(" AFTER ")
		builder.WriteString(a.After.String(level))
	}
	return builder.String()
}

func (a *AlterTableAddIndex) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Index.Accept(visitor); err != nil {
		return err
	}
	if a.After != nil {
		if err := a.After.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableAddIndex(a)
}

type AlterTableDropColumn struct {
	DropPos    Pos
	ColumnName *NestedIdentifier
	IfExists   bool
}

func (a *AlterTableDropColumn) Pos() Pos {
	return a.DropPos
}

func (a *AlterTableDropColumn) End() Pos {
	return a.ColumnName.End()
}

func (a *AlterTableDropColumn) AlterType() string {
	return "DROP_COLUMN"
}

func (a *AlterTableDropColumn) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP COLUMN ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(a.ColumnName.String(level))
	return builder.String()
}

func (a *AlterTableDropColumn) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.ColumnName.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableDropColumn(a)
}

type AlterTableDropIndex struct {
	DropPos   Pos
	IndexName *NestedIdentifier
	IfExists  bool
}

func (a *AlterTableDropIndex) Pos() Pos {
	return a.DropPos
}

func (a *AlterTableDropIndex) End() Pos {
	return a.IndexName.End()
}

func (a *AlterTableDropIndex) AlterType() string {
	return "DROP_INDEX"
}

func (a *AlterTableDropIndex) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP INDEX ")
	builder.WriteString(a.IndexName.String(level))
	if a.IfExists {
		builder.WriteString(" IF EXISTS")
	}
	return builder.String()
}

func (a *AlterTableDropIndex) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.IndexName.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableDropIndex(a)
}

type AlterTableRemoveTTL struct {
	RemovePos    Pos
	StatementEnd Pos
}

func (a *AlterTableRemoveTTL) Pos() Pos {
	return a.RemovePos
}

func (a *AlterTableRemoveTTL) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableRemoveTTL) AlterType() string {
	return "REMOVE_TTL"
}

func (a *AlterTableRemoveTTL) String(level int) string {
	return "REMOVE TTL"
}

func (a *AlterTableRemoveTTL) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	return visitor.VisitAlterTableRemoveTTL(a)
}

type AlterTableClearColumn struct {
	ClearPos     Pos
	StatementEnd Pos

	IfExists      bool
	ColumnName    *NestedIdentifier
	PartitionExpr *PartitionExpr
}

func (a *AlterTableClearColumn) Pos() Pos {
	return a.ClearPos
}

func (a *AlterTableClearColumn) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableClearColumn) AlterType() string {
	return "CLEAR_COLUMN"
}

func (a *AlterTableClearColumn) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CLEAR COLUMN ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(a.ColumnName.String(level))
	if a.PartitionExpr != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString("IN ")
		builder.WriteString(a.PartitionExpr.String(level))
	}

	return builder.String()
}

func (a *AlterTableClearColumn) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.ColumnName.Accept(visitor); err != nil {
		return err
	}
	if a.PartitionExpr != nil {
		if err := a.PartitionExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableClearColumn(a)
}

type AlterTableClearIndex struct {
	ClearPos     Pos
	StatementEnd Pos

	IfExists      bool
	IndexName     *NestedIdentifier
	PartitionExpr *PartitionExpr
}

func (a *AlterTableClearIndex) Pos() Pos {
	return a.ClearPos
}

func (a *AlterTableClearIndex) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableClearIndex) AlterType() string {
	return "CLEAR_INDEX"
}

func (a *AlterTableClearIndex) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CLEAR INDEX ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(a.IndexName.String(level + 1))
	if a.PartitionExpr != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString("IN ")
		builder.WriteString(a.PartitionExpr.String(level))
	}

	return builder.String()
}

func (a *AlterTableClearIndex) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.IndexName.Accept(visitor); err != nil {
		return err
	}
	if a.PartitionExpr != nil {
		if err := a.PartitionExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableClearIndex(a)
}

type AlterTableRenameColumn struct {
	RenamePos Pos

	IfExists      bool
	OldColumnName *NestedIdentifier
	NewColumnName *NestedIdentifier
}

func (a *AlterTableRenameColumn) Pos() Pos {
	return a.RenamePos
}

func (a *AlterTableRenameColumn) End() Pos {
	return a.NewColumnName.End()
}

func (a *AlterTableRenameColumn) AlterType() string {
	return "RENAME_COLUMN"
}

func (a *AlterTableRenameColumn) String(level int) string {
	var builder strings.Builder
	builder.WriteString("RENAME COLUMN ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(a.OldColumnName.String(level))
	builder.WriteString(" TO ")
	builder.WriteString(a.NewColumnName.String(level))
	return builder.String()
}

func (a *AlterTableRenameColumn) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.OldColumnName.Accept(visitor); err != nil {
		return err
	}
	if err := a.NewColumnName.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableRenameColumn(a)
}

type AlterTableModifyTTL struct {
	ModifyPos    Pos
	StatementEnd Pos
	TTL          *TTLExpr
}

func (a *AlterTableModifyTTL) Pos() Pos {
	return a.ModifyPos
}

func (a *AlterTableModifyTTL) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableModifyTTL) AlterType() string {
	return "MODIFY_TTL"
}

func (a *AlterTableModifyTTL) String(level int) string {
	var builder strings.Builder
	builder.WriteString("MODIFY ")
	builder.WriteString("TTL ")
	builder.WriteString(a.TTL.String(level))
	return builder.String()
}

func (a *AlterTableModifyTTL) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.TTL.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableModifyTTL(a)
}

type AlterTableModifyColumn struct {
	ModifyPos    Pos
	StatementEnd Pos

	IfExists           bool
	Column             *Column
	RemovePropertyType *RemovePropertyType
}

func (a *AlterTableModifyColumn) Pos() Pos {
	return a.ModifyPos
}

func (a *AlterTableModifyColumn) End() Pos {
	return a.StatementEnd
}

func (a *AlterTableModifyColumn) AlterType() string {
	return "MODIFY_COLUMN"
}

func (a *AlterTableModifyColumn) String(level int) string {
	var builder strings.Builder
	builder.WriteString("MODIFY COLUMN ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(a.Column.String(level))
	if a.RemovePropertyType != nil {
		builder.WriteString(a.RemovePropertyType.String(level))
	}
	return builder.String()
}

func (a *AlterTableModifyColumn) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Column.Accept(visitor); err != nil {
		return err
	}
	if a.RemovePropertyType != nil {
		if err := a.RemovePropertyType.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterTableModifyColumn(a)
}

type AlterTableReplacePartition struct {
	ReplacePos Pos
	Partition  *PartitionExpr
	Table      *TableIdentifier
}

func (a *AlterTableReplacePartition) Pos() Pos {
	return a.ReplacePos
}

func (a *AlterTableReplacePartition) End() Pos {
	return a.Table.End()
}

func (a *AlterTableReplacePartition) AlterType() string {
	return "REPLACE_PARTITION"
}

func (a *AlterTableReplacePartition) String(level int) string {
	var builder strings.Builder
	builder.WriteString("REPLACE ")
	builder.WriteString(a.Partition.String(level))
	builder.WriteString(" FROM ")
	builder.WriteString(a.Table.String(level))
	return builder.String()
}

func (a *AlterTableReplacePartition) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Partition.Accept(visitor); err != nil {
		return err
	}
	if err := a.Table.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAlterTableReplacePartition(a)
}

type RemovePropertyType struct {
	RemovePos Pos

	PropertyType Expr
}

func (a *RemovePropertyType) Pos() Pos {
	return a.RemovePos
}

func (a *RemovePropertyType) End() Pos {
	return a.PropertyType.End()
}

func (a *RemovePropertyType) String(level int) string {
	var builder strings.Builder
	builder.WriteString(" REMOVE ")
	builder.WriteString(a.PropertyType.String(level))
	return builder.String()
}

func (a *RemovePropertyType) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.PropertyType.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitRemovePropertyType(a)
}

type TableIndex struct {
	IndexPos Pos

	Name        *NestedIdentifier
	ColumnExpr  Expr
	ColumnType  Expr
	Granularity *NumberLiteral
}

func (a *TableIndex) Pos() Pos {
	return a.IndexPos
}

func (a *TableIndex) End() Pos {
	return a.Granularity.End()
}

func (a *TableIndex) String(level int) string {
	var builder strings.Builder
	builder.WriteString(a.Name.String(0))
	builder.WriteString(a.ColumnExpr.String(level))
	builder.WriteString("TYPE")
	builder.WriteByte(' ')
	builder.WriteString(a.ColumnType.String(level))
	builder.WriteByte(' ')
	builder.WriteString("GRANULARITY")
	builder.WriteByte(' ')
	builder.WriteString(a.Granularity.String(level))
	return builder.String()
}

func (a *TableIndex) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Name.Accept(visitor); err != nil {
		return err
	}
	if err := a.ColumnExpr.Accept(visitor); err != nil {
		return err
	}
	if err := a.ColumnType.Accept(visitor); err != nil {
		return err
	}
	if err := a.Granularity.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTableIndex(a)
}

type Ident struct {
	Name      string
	QuoteType int
	NamePos   Pos
	NameEnd   Pos
}

func (i *Ident) Pos() Pos {
	return i.NamePos
}

func (i *Ident) End() Pos {
	return i.NameEnd
}

func (i *Ident) String(int) string {
	if i.QuoteType == BackTicks {
		return "`" + i.Name + "`"
	} else if i.QuoteType == DoubleQuote {
		return `"` + i.Name + `"`
	}
	return i.Name
}

func (i *Ident) Accept(visitor ASTVisitor) error {
	visitor.enter(i)
	defer visitor.leave(i)
	return visitor.VisitIdent(i)
}

type UUID struct {
	Value *StringLiteral
}

func (u *UUID) Pos() Pos {
	return u.Value.LiteralPos
}

func (u *UUID) End() Pos {
	return u.Value.LiteralEnd
}

func (u *UUID) String(level int) string {
	return "UUID " + u.Value.String(level)
}

func (u *UUID) Accept(visitor ASTVisitor) error {
	visitor.enter(u)
	defer visitor.leave(u)
	return visitor.VisitUUID(u)
}

type CreateDatabase struct {
	CreatePos    Pos // position of CREATE keyword
	StatementEnd Pos
	Name         Expr
	IfNotExists  bool // true if 'IF NOT EXISTS' is specified
	OnCluster    *OnClusterExpr
	Engine       *EngineExpr
}

func (c *CreateDatabase) Pos() Pos {
	return c.CreatePos
}

func (c *CreateDatabase) End() Pos {
	return c.StatementEnd
}

func (c *CreateDatabase) Type() string {
	return "DATABASE"
}

func (c *CreateDatabase) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE DATABASE ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.Name.String(level))
	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}
	if c.Engine != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.Engine.String(level))
	}
	return builder.String()
}

func (c *CreateDatabase) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Engine != nil {
		if err := c.Engine.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateDatabase(c)
}

type CreateTable struct {
	CreatePos    Pos // position of CREATE|ATTACH keyword
	StatementEnd Pos
	Name         *TableIdentifier
	IfNotExists  bool
	UUID         *UUID
	OnCluster    *OnClusterExpr
	TableSchema  *TableSchemaExpr
	Engine       *EngineExpr
	SubQuery     *SubQueryExpr
	HasTemporary bool
}

func (c *CreateTable) Pos() Pos {
	return c.CreatePos
}

func (c *CreateTable) End() Pos {
	return c.StatementEnd
}

func (c *CreateTable) Type() string {
	return "CREATE TABLE"
}

func (c *CreateTable) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE")
	if c.HasTemporary {
		builder.WriteString(" TEMPORARY")
	}
	builder.WriteString(" TABLE ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.Name.String(level))
	if c.UUID != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.UUID.String(level))
	}
	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}
	if c.TableSchema != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.TableSchema.String(level))
	}
	if c.Engine != nil {
		builder.WriteString(c.Engine.String(level))
	}
	if c.SubQuery != nil {
		builder.WriteString(c.SubQuery.String(level))
	}
	return builder.String()
}

func (c *CreateTable) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if c.UUID != nil {
		if err := c.UUID.Accept(visitor); err != nil {
			return err
		}
	}
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if c.TableSchema != nil {
		if err := c.TableSchema.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Engine != nil {
		if err := c.Engine.Accept(visitor); err != nil {
			return err
		}
	}
	if c.SubQuery != nil {
		if err := c.SubQuery.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateTable(c)
}

type CreateMaterializedView struct {
	CreatePos    Pos // position of CREATE|ATTACH keyword
	StatementEnd Pos
	Name         *TableIdentifier
	IfNotExists  bool
	OnCluster    *OnClusterExpr
	Engine       *EngineExpr
	Destination  *DestinationExpr
	SubQuery     *SubQueryExpr
	Populate     bool
}

func (c *CreateMaterializedView) Pos() Pos {
	return c.CreatePos
}

func (c *CreateMaterializedView) End() Pos {
	return c.StatementEnd
}

func (c *CreateMaterializedView) Type() string {
	return "MATERIALIZED_VIEW"
}

func (c *CreateMaterializedView) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE MATERIALIZED VIEW ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.Name.String(level))
	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}
	if c.Engine != nil {
		builder.WriteString(c.Engine.String(level))
	}
	if c.Destination != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.Destination.String(level))
		if c.Destination.TableSchema != nil {
			builder.WriteString(NewLine(level))
			// level + 1 to add an indent for table schema
			builder.WriteString(c.Destination.TableSchema.String(level + 1))
		}
	}
	if c.Populate {
		builder.WriteString(" POPULATE ")
	}
	if c.SubQuery != nil {
		builder.WriteString(c.SubQuery.String(level))
	}
	return builder.String()
}

func (c *CreateMaterializedView) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Engine != nil {
		if err := c.Engine.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Destination != nil {
		if err := c.Destination.Accept(visitor); err != nil {
			return err
		}
		if c.Destination.TableSchema != nil {
			if err := c.Destination.TableSchema.Accept(visitor); err != nil {
				return err
			}
		}
	}
	if c.SubQuery != nil {
		if err := c.SubQuery.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateMaterializedView(c)
}

type CreateView struct {
	CreatePos    Pos // position of CREATE|ATTACH keyword
	StatementEnd Pos
	Name         *TableIdentifier
	IfNotExists  bool
	UUID         *UUID
	OnCluster    *OnClusterExpr
	TableSchema  *TableSchemaExpr
	SubQuery     *SubQueryExpr
}

func (c *CreateView) Pos() Pos {
	return c.CreatePos
}

func (c *CreateView) End() Pos {
	return c.StatementEnd
}

func (c *CreateView) Type() string {
	return "VIEW"
}

func (c *CreateView) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE VIEW ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.Name.String(level))
	if c.UUID != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.UUID.String(level))
	}

	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}

	if c.TableSchema != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.TableSchema.String(level))
	}

	if c.SubQuery != nil {
		builder.WriteString(c.SubQuery.String(level))
	}
	return builder.String()
}

func (c *CreateView) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	if c.UUID != nil {
		if err := c.UUID.Accept(visitor); err != nil {
			return err
		}
	}
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if c.TableSchema != nil {
		if err := c.TableSchema.Accept(visitor); err != nil {
			return err
		}
	}
	if c.SubQuery != nil {
		if err := c.SubQuery.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateView(c)
}

type CreateFunction struct {
	CreatePos    Pos
	IfNotExists  bool
	FunctionName *Ident
	OnCluster    *OnClusterExpr
	Params       *ParamExprList
	Expr         Expr
}

func (c *CreateFunction) Type() string {
	return "FUNCTION"
}

func (c *CreateFunction) Pos() Pos {
	return c.CreatePos
}

func (c *CreateFunction) End() Pos {
	return c.Expr.End()
}

func (c *CreateFunction) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE FUNCTION ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.FunctionName.String(level))
	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}
	builder.WriteString(" AS ")
	builder.WriteString(c.Params.String(level))
	builder.WriteString(" -> ")
	builder.WriteString(c.Expr.String(level))
	return builder.String()
}

func (c *CreateFunction) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.FunctionName.Accept(visitor); err != nil {
		return err
	}
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if err := c.Params.Accept(visitor); err != nil {
		return err
	}
	if err := c.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitCreateFunction(c)
}

type RoleName struct {
	Name      Expr
	Scope     *StringLiteral
	OnCluster *OnClusterExpr
}

func (r *RoleName) Pos() Pos {
	return r.Name.Pos()
}

func (r *RoleName) End() Pos {
	if r.Scope != nil {
		return r.Scope.End()
	}
	if r.OnCluster != nil {
		return r.OnCluster.End()
	}
	return r.Name.End()
}

func (r *RoleName) String(level int) string {
	var builder strings.Builder
	builder.WriteString(r.Name.String(level))
	if r.Scope != nil {
		builder.WriteString("@")
		builder.WriteString(r.Scope.String(level))
	}
	if r.OnCluster != nil {
		builder.WriteString(" ON ")
		builder.WriteString(r.OnCluster.String(level))
	}
	return builder.String()
}

func (r *RoleName) Accept(visitor ASTVisitor) error {
	visitor.enter(r)
	defer visitor.leave(r)
	if err := r.Name.Accept(visitor); err != nil {
		return err
	}
	if r.Scope != nil {
		if err := r.Scope.Accept(visitor); err != nil {
			return err
		}
	}
	if r.OnCluster != nil {
		if err := r.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitRoleName(r)
}

type SettingPair struct {
	Name  *Ident
	Value Expr
}

func (s *SettingPair) Pos() Pos {
	return s.Name.NamePos
}

func (s *SettingPair) End() Pos {
	return s.Value.End()
}

func (s *SettingPair) String(level int) string {
	var builder strings.Builder
	builder.WriteString(s.Name.String(level))
	if s.Value != nil {
		builder.WriteByte(' ')
		builder.WriteString(s.Value.String(level))
	}
	return builder.String()
}

func (s *SettingPair) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Name.Accept(visitor); err != nil {
		return err
	}
	if s.Value != nil {
		if err := s.Value.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSettingPair(s)
}

type RoleSetting struct {
	SettingPairs []*SettingPair
	Modifier     *Ident
}

func (r *RoleSetting) Pos() Pos {
	if len(r.SettingPairs) > 0 {
		return r.SettingPairs[0].Pos()
	}
	return r.Modifier.NamePos
}

func (r *RoleSetting) End() Pos {
	if r.Modifier != nil {
		return r.Modifier.NameEnd
	}
	return r.SettingPairs[len(r.SettingPairs)-1].End()
}

func (r *RoleSetting) String(level int) string {
	var builder strings.Builder
	for i, settingPair := range r.SettingPairs {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(settingPair.String(level))
	}
	if r.Modifier != nil {
		builder.WriteString(" ")
		builder.WriteString(r.Modifier.String(level))
	}
	return builder.String()
}

func (r *RoleSetting) Accept(visitor ASTVisitor) error {
	visitor.enter(r)
	defer visitor.leave(r)
	for _, settingPair := range r.SettingPairs {
		if err := settingPair.Accept(visitor); err != nil {
			return err
		}
	}
	if r.Modifier != nil {
		if err := r.Modifier.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitRoleSetting(r)
}

type CreateRole struct {
	CreatePos         Pos
	StatementEnd      Pos
	IfNotExists       bool
	OrReplace         bool
	RoleNames         []*RoleName
	AccessStorageType *Ident
	Settings          []*RoleSetting
}

func (c *CreateRole) Pos() Pos {
	return c.CreatePos
}

func (c *CreateRole) End() Pos {
	return c.StatementEnd
}

func (c *CreateRole) Type() string {
	return "ROLE"
}

func (c *CreateRole) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE ROLE ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	if c.OrReplace {
		builder.WriteString("OR REPLACE ")
	}
	for i, roleName := range c.RoleNames {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(roleName.String(level))
	}
	if c.AccessStorageType != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString("IN ")
		builder.WriteString(c.AccessStorageType.String(level))
	}
	if len(c.Settings) > 0 {
		builder.WriteString(" SETTINGS ")
		for i, setting := range c.Settings {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(setting.String(level))
		}
	}
	return builder.String()
}

func (c *CreateRole) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	for _, roleName := range c.RoleNames {
		if err := roleName.Accept(visitor); err != nil {
			return err
		}
	}
	if c.AccessStorageType != nil {
		if err := c.AccessStorageType.Accept(visitor); err != nil {
			return err
		}
	}
	for _, setting := range c.Settings {
		if err := setting.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateRole(c)
}

type AlterRole struct {
	AlterPos        Pos
	StatementEnd    Pos
	IfExists        bool
	RoleRenamePairs []*RoleRenamePair
	Settings        []*RoleSetting
}

func (a *AlterRole) Pos() Pos {
	return a.AlterPos
}

func (a *AlterRole) End() Pos {
	return a.StatementEnd
}

func (a *AlterRole) Type() string {
	return "ROLE"
}

func (a *AlterRole) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ALTER ROLE ")
	if a.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	for i, roleRenamePair := range a.RoleRenamePairs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(roleRenamePair.String(level))
	}
	if len(a.Settings) > 0 {
		builder.WriteString(" SETTINGS ")
		for i, setting := range a.Settings {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(setting.String(level))
		}
	}
	return builder.String()
}

func (a *AlterRole) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	for _, roleRenamePair := range a.RoleRenamePairs {
		if err := roleRenamePair.Accept(visitor); err != nil {
			return err
		}
	}
	for _, setting := range a.Settings {
		if err := setting.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitAlterRole(a)
}

type RoleRenamePair struct {
	RoleName     *RoleName
	NewName      Expr
	StatementEnd Pos
}

func (r *RoleRenamePair) Pos() Pos {
	return r.RoleName.Pos()
}

func (r *RoleRenamePair) End() Pos {
	return r.StatementEnd
}

func (r *RoleRenamePair) String(level int) string {
	var builder strings.Builder
	builder.WriteString(r.RoleName.String(level))
	if r.NewName != nil {
		builder.WriteString(" RENAME TO ")
		builder.WriteString(r.NewName.String(level))
	}
	return builder.String()
}

func (r *RoleRenamePair) Accept(visitor ASTVisitor) error {
	visitor.enter(r)
	defer visitor.leave(r)
	if err := r.RoleName.Accept(visitor); err != nil {
		return err
	}
	if r.NewName != nil {
		if err := r.NewName.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitRoleRenamePair(r)
}

type DestinationExpr struct {
	ToPos           Pos
	TableIdentifier *TableIdentifier
	TableSchema     *TableSchemaExpr
}

func (d *DestinationExpr) Pos() Pos {
	return d.ToPos
}

func (d *DestinationExpr) End() Pos {
	return d.TableIdentifier.End()
}

func (d *DestinationExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("TO ")
	builder.WriteString(d.TableIdentifier.String(level))
	return builder.String()
}

func (d *DestinationExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if err := d.TableIdentifier.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitDestinationExpr(d)
}

type ConstraintExpr struct {
	ConstraintPos Pos
	Constraint    *Ident
	Expr          Expr
}

func (c *ConstraintExpr) Pos() Pos {
	return c.ConstraintPos
}

func (c *ConstraintExpr) End() Pos {
	return c.Expr.End()
}

func (c *ConstraintExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(c.Constraint.String(level))
	builder.WriteByte(' ')
	builder.WriteString(c.Expr.String(level))
	return builder.String()
}

func (c *ConstraintExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Constraint.Accept(visitor); err != nil {
		return err
	}
	if err := c.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitConstraintExpr(c)
}

type NullLiteral struct {
	NullPos Pos
}

func (n *NullLiteral) Pos() Pos {
	return n.NullPos
}

func (n *NullLiteral) End() Pos {
	return n.NullPos + 4
}

func (n *NullLiteral) String(int) string {
	return "NULL"
}

func (n *NullLiteral) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	return visitor.VisitNullLiteral(n)
}

type NotNullLiteral struct {
	NotPos      Pos
	NullLiteral *NullLiteral
}

func (n *NotNullLiteral) Pos() Pos {
	return n.NotPos
}

func (n *NotNullLiteral) End() Pos {
	return n.NullLiteral.End()
}

func (n *NotNullLiteral) String(int) string {
	return "NOT NULL"
}

func (n *NotNullLiteral) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.NullLiteral.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitNotNullLiteral(n)
}

type NestedIdentifier struct {
	Ident    *Ident
	DotIdent *Ident
}

func (n *NestedIdentifier) Pos() Pos {
	return n.Ident.Pos()
}

func (n *NestedIdentifier) End() Pos {
	if n.DotIdent != nil {
		return n.DotIdent.End()
	}
	return n.Ident.End()
}

func (n *NestedIdentifier) String(int) string {
	if n.DotIdent != nil {
		return n.Ident.String(0) + "." + n.DotIdent.String(0)
	}
	return n.Ident.String(0)
}

func (n *NestedIdentifier) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Ident.Accept(visitor); err != nil {
		return err
	}
	if n.DotIdent != nil {
		if err := n.DotIdent.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitNestedIdentifier(n)
}

type ColumnIdentifier struct {
	Database *Ident
	Table    *Ident
	Column   *Ident
}

func (c *ColumnIdentifier) Pos() Pos {
	if c.Database != nil {
		return c.Database.NamePos
	} else if c.Table != nil {
		return c.Table.NamePos
	} else {
		return c.Column.NamePos
	}
}

func (c *ColumnIdentifier) End() Pos {
	return c.Column.NameEnd
}

func (c *ColumnIdentifier) String(int) string {
	if c.Database != nil {
		return c.Database.String(0) + "." + c.Table.String(0) + "." + c.Column.String(0)
	} else if c.Table != nil {
		return c.Table.String(0) + "." + c.Column.String(0)
	} else {
		return c.Column.String(0)
	}
}

func (c *ColumnIdentifier) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if c.Database != nil {
		if err := c.Database.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Table != nil {
		if err := c.Table.Accept(visitor); err != nil {
			return err
		}
	}
	if err := c.Column.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitColumnIdentifier(c)
}

type TableIdentifier struct {
	Database *Ident
	Table    *Ident
}

func (t *TableIdentifier) Pos() Pos {
	if t.Database != nil {
		return t.Database.Pos()
	}
	return t.Table.Pos()
}

func (t *TableIdentifier) End() Pos {
	return t.Table.End()
}

func (t *TableIdentifier) String(int) string {
	if t.Database != nil {
		return t.Database.String(0) + "." + t.Table.String(0)
	}
	return t.Table.String(0)
}

func (t *TableIdentifier) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if t.Database != nil {
		if err := t.Database.Accept(visitor); err != nil {
			return err
		}
	}
	if err := t.Table.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTableIdentifier(t)
}

type TableSchemaExpr struct {
	SchemaPos     Pos
	SchemaEnd     Pos
	Columns       []Expr
	AliasTable    *TableIdentifier
	TableFunction *TableFunctionExpr
}

func (t *TableSchemaExpr) Pos() Pos {
	return t.SchemaPos
}

func (t *TableSchemaExpr) End() Pos {
	return t.SchemaEnd
}

func (t *TableSchemaExpr) String(level int) string {
	var builder strings.Builder
	if len(t.Columns) > 0 {
		builder.WriteString("(")
		for i, column := range t.Columns {
			if i > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(NewLine(level + 1))
			builder.WriteString(column.String(level))
		}
		builder.WriteString(NewLine(level - 1))
		builder.WriteByte(')')
	}
	if t.AliasTable != nil {
		builder.WriteString(" AS ")
		builder.WriteString(t.AliasTable.String(level))
	}
	if t.TableFunction != nil {
		builder.WriteByte(' ')
		builder.WriteString(t.TableFunction.String(level))
	}
	return builder.String()
}

func (t *TableSchemaExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	for _, column := range t.Columns {
		if err := column.Accept(visitor); err != nil {
			return err
		}
	}
	if t.AliasTable != nil {
		if err := t.AliasTable.Accept(visitor); err != nil {
			return err
		}
	}
	if t.TableFunction != nil {
		if err := t.TableFunction.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTableSchemaExpr(t)
}

type TableArgListExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Args          []Expr
}

func (t *TableArgListExpr) Pos() Pos {
	return t.LeftParenPos
}

func (t *TableArgListExpr) End() Pos {
	return t.RightParenPos
}

func (t *TableArgListExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteByte('(')
	for i, arg := range t.Args {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(arg.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (t *TableArgListExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	for _, arg := range t.Args {
		if err := arg.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTableArgListExpr(t)
}

type TableFunctionExpr struct {
	Name Expr
	Args *TableArgListExpr
}

func (t *TableFunctionExpr) Pos() Pos {
	return t.Name.Pos()
}

func (t *TableFunctionExpr) End() Pos {
	return t.Args.End()
}

func (t *TableFunctionExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(t.Name.String(level))
	builder.WriteString(t.Args.String(level))
	return builder.String()
}

func (t *TableFunctionExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.Name.Accept(visitor); err != nil {
		return err
	}
	if err := t.Args.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTableFunctionExpr(t)
}

type OnClusterExpr struct {
	OnPos Pos
	Expr  Expr
}

func (o *OnClusterExpr) Pos() Pos {
	return o.OnPos
}

func (o *OnClusterExpr) End() Pos {
	return o.Expr.End()
}

func (o *OnClusterExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ON CLUSTER ")
	builder.WriteString(o.Expr.String(level + 1))
	return builder.String()
}

func (o *OnClusterExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	if err := o.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitOnClusterExpr(o)
}

type DefaultExpr struct {
	DefaultPos Pos
	Expr       Expr
}

func (d *DefaultExpr) Pos() Pos {
	return d.DefaultPos
}

func (d *DefaultExpr) End() Pos {
	return d.Expr.End()
}

func (d *DefaultExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DEFAULT ")
	builder.WriteString(d.Expr.String(level + 1))
	return builder.String()
}

func (d *DefaultExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if err := d.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitDefaultExpr(d)
}

type PartitionExpr struct {
	PartitionPos Pos
	Expr         Expr
	ID           *StringLiteral
	All          bool
}

func (p *PartitionExpr) Pos() Pos {
	return p.PartitionPos
}

func (p *PartitionExpr) End() Pos {
	if p.ID != nil {
		return p.ID.LiteralEnd
	}
	return p.Expr.End()
}

func (p *PartitionExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("PARTITION ")
	if p.ID != nil {
		builder.WriteString(p.ID.String(level))
	} else if p.All {
		builder.WriteString("ALL")
	} else {
		builder.WriteString(p.Expr.String(level))
	}
	return builder.String()
}

func (p *PartitionExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(p)
	defer visitor.leave(p)
	if p.Expr != nil {
		if err := p.Expr.Accept(visitor); err != nil {
			return err
		}
	}
	if p.ID != nil {
		if err := p.ID.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitPartitionExpr(p)
}

type PartitionByExpr struct {
	PartitionPos Pos
	Expr         Expr
}

func (p *PartitionByExpr) Pos() Pos {
	return p.PartitionPos
}

func (p *PartitionByExpr) End() Pos {
	return p.Expr.End()
}

func (p *PartitionByExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("PARTITION BY ")
	builder.WriteString(p.Expr.String(level))
	return builder.String()
}

func (p *PartitionByExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(p)
	defer visitor.leave(p)
	if err := p.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitPartitionByExpr(p)
}

type PrimaryKeyExpr struct {
	PrimaryPos Pos
	Expr       Expr
}

func (p *PrimaryKeyExpr) Pos() Pos {
	return p.PrimaryPos
}

func (p *PrimaryKeyExpr) End() Pos {
	return p.Expr.End()
}

func (p *PrimaryKeyExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("PRIMARY KEY ")
	builder.WriteString(p.Expr.String(level))
	return builder.String()
}

func (p *PrimaryKeyExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(p)
	defer visitor.leave(p)
	if err := p.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitPrimaryKeyExpr(p)
}

type SampleByExpr struct {
	SamplePos Pos
	Expr      Expr
}

func (s *SampleByExpr) Pos() Pos {
	return s.SamplePos
}

func (s *SampleByExpr) End() Pos {
	return s.Expr.End()
}

func (s *SampleByExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("SAMPLE BY ")
	builder.WriteString(s.Expr.String(level))
	return builder.String()
}

func (s *SampleByExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitSampleByExpr(s)
}

type TTLExpr struct {
	TTLPos Pos
	Expr   Expr
}

func (t *TTLExpr) Pos() Pos {
	return t.TTLPos
}

func (t *TTLExpr) End() Pos {
	return t.Expr.End()
}

func (t *TTLExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(t.Expr.String(level))
	return builder.String()
}

func (t *TTLExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTTLExpr(t)
}

type TTLExprList struct {
	TTLPos  Pos
	ListEnd Pos
	Items   []*TTLExpr
}

func (t *TTLExprList) Pos() Pos {
	return t.TTLPos
}

func (t *TTLExprList) End() Pos {
	return t.ListEnd
}

func (t *TTLExprList) String(level int) string {
	var builder strings.Builder
	builder.WriteString("TTL ")
	for i, item := range t.Items {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(item.String(level))
	}
	return builder.String()
}

func (t *TTLExprList) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	for _, item := range t.Items {
		if err := item.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTTLExprList(t)
}

type OrderByExpr struct {
	OrderPos  Pos
	Expr      Expr
	Direction OrderDirection
}

func (o *OrderByExpr) Pos() Pos {
	return o.OrderPos
}

func (o *OrderByExpr) End() Pos {
	return o.Expr.End()
}

func (o *OrderByExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(o.Expr.String(level))
	if o.Direction != OrderDirectionNone {
		builder.WriteByte(' ')
		builder.WriteString(string(o.Direction))
	}
	return builder.String()
}

func (o *OrderByExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	if err := o.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitOrderByExpr(o)
}

type OrderByListExpr struct {
	OrderPos Pos
	ListEnd  Pos
	Items    []Expr
}

func (o *OrderByListExpr) Pos() Pos {
	return o.OrderPos
}

func (o *OrderByListExpr) End() Pos {
	return o.ListEnd
}

func (o *OrderByListExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ORDER BY ")
	for i, item := range o.Items {
		builder.WriteString(item.String(level))
		if i != len(o.Items)-1 {
			builder.WriteByte(',')
			builder.WriteByte(' ')
		}
	}
	return builder.String()
}

func (o *OrderByListExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	for _, item := range o.Items {
		if err := item.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitOrderByListExpr(o)
}

type SettingsExpr struct {
	SettingsPos Pos
	Name        *Ident
	Expr        Expr
}

func (s *SettingsExpr) Pos() Pos {
	return s.SettingsPos
}

func (s *SettingsExpr) End() Pos {
	return s.Expr.End()
}

func (s *SettingsExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(s.Name.String(level))
	builder.WriteByte('=')
	builder.WriteString(s.Expr.String(level))
	return builder.String()
}

func (s *SettingsExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Name.Accept(visitor); err != nil {
		return err
	}
	if err := s.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitSettingsExpr(s)
}

type SettingsExprList struct {
	SettingsPos Pos
	ListEnd     Pos
	Items       []*SettingsExpr
}

func (s *SettingsExprList) Pos() Pos {
	return s.SettingsPos
}

func (s *SettingsExprList) End() Pos {
	return s.ListEnd
}

func (s *SettingsExprList) String(level int) string {
	var builder strings.Builder
	builder.WriteString("SETTINGS ")
	for i, item := range s.Items {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(item.String(level))
	}
	return builder.String()
}

func (s *SettingsExprList) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	for _, item := range s.Items {
		if err := item.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSettingsExprList(s)
}

type ParamExprList struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Items         *ColumnExprList
	ColumnArgList *ColumnArgList
}

func (f *ParamExprList) Pos() Pos {
	return f.LeftParenPos
}

func (f *ParamExprList) End() Pos {
	return f.RightParenPos
}

func (f *ParamExprList) String(level int) string {
	var builder strings.Builder
	builder.WriteString("(")
	for i, item := range f.Items.Items {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(item.String(level))
	}
	builder.WriteString(")")
	return builder.String()
}

func (f *ParamExprList) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Items.Accept(visitor); err != nil {
		return err
	}
	if f.ColumnArgList != nil {
		if err := f.ColumnArgList.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitParamExprList(f)
}

type ArrayParamList struct {
	LeftBracketPos  Pos
	RightBracketPos Pos
	Items           *ColumnExprList
}

func (a *ArrayParamList) Pos() Pos {
	return a.LeftBracketPos
}

func (a *ArrayParamList) End() Pos {
	return a.RightBracketPos
}

func (a *ArrayParamList) String(level int) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i, item := range a.Items.Items {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(item.String(level))
	}
	builder.WriteString("]")
	return builder.String()
}

func (a *ArrayParamList) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Items.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitArrayParamList(a)
}

type ObjectParams struct {
	Object Expr
	Params *ArrayParamList
}

func (o *ObjectParams) Pos() Pos {
	return o.Object.Pos()
}

func (o *ObjectParams) End() Pos {
	return o.Params.End()
}

func (o *ObjectParams) String(level int) string {
	var builder strings.Builder
	builder.WriteString(o.Object.String(level))
	builder.WriteString(o.Params.String(level))
	return builder.String()
}

func (o *ObjectParams) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	if err := o.Object.Accept(visitor); err != nil {
		return err
	}
	if err := o.Params.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitObjectParams(o)
}

type FunctionExpr struct {
	Name   *Ident
	Params *ParamExprList
}

func (f *FunctionExpr) Pos() Pos {
	return f.Name.NamePos
}

func (f *FunctionExpr) End() Pos {
	return f.Params.RightParenPos
}

func (f *FunctionExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(f.Name.String(level))
	builder.WriteString(f.Params.String(level))
	return builder.String()
}

func (f *FunctionExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Name.Accept(visitor); err != nil {
		return err
	}
	if err := f.Params.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitFunctionExpr(f)
}

type WindowFunctionExpr struct {
	Function *FunctionExpr
	OverPos  Pos
	OverExpr Expr
}

func (w *WindowFunctionExpr) Pos() Pos {
	return w.Function.Pos()
}

func (w *WindowFunctionExpr) End() Pos {
	return w.OverExpr.End()
}

func (w *WindowFunctionExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(w.Function.String(level))
	builder.WriteString(" OVER ")
	builder.WriteString(w.OverExpr.String(level))
	return builder.String()
}

func (w *WindowFunctionExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if err := w.Function.Accept(visitor); err != nil {
		return err
	}
	if err := w.OverExpr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWindowFunctionExpr(w)
}

type Column struct {
	NamePos   Pos
	ColumnEnd Pos
	Name      *NestedIdentifier
	Type      Expr
	NotNull   *NotNullLiteral
	Nullable  *NullLiteral

	Property Expr

	Codec *CompressionCodec
	TTL   Expr

	Comment          *StringLiteral
	CompressionCodec *Ident
}

func (c *Column) Pos() Pos {
	return c.Name.Pos()
}

func (c *Column) End() Pos {
	return c.ColumnEnd
}

func (c *Column) String(level int) string {
	var builder strings.Builder
	builder.WriteString(c.Name.String(level))
	if c.Type != nil {
		builder.WriteByte(' ')
		builder.WriteString(c.Type.String(level))
	}
	if c.NotNull != nil {
		builder.WriteString(" NOT NULL")
	} else if c.Nullable != nil {
		builder.WriteString(" NULL")
	}
	if c.Property != nil {
		builder.WriteByte(' ')
		builder.WriteString(c.Property.String(level))
	}
	if c.Codec != nil {
		builder.WriteByte(' ')
		builder.WriteString(c.Codec.String(level))
	}
	if c.TTL != nil {
		builder.WriteByte(' ')
		builder.WriteString(c.TTL.String(level))
	}
	if c.Comment != nil {
		builder.WriteString(" COMMENT ")
		builder.WriteString(c.Comment.String(level))
	}
	return builder.String()
}

func (c *Column) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	if c.Type != nil {
		if err := c.Type.Accept(visitor); err != nil {
			return err
		}
	}
	if c.NotNull != nil {
		if err := c.NotNull.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Nullable != nil {
		if err := c.Nullable.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Property != nil {
		if err := c.Property.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Codec != nil {
		if err := c.Codec.Accept(visitor); err != nil {
			return err
		}
	}
	if c.TTL != nil {
		if err := c.TTL.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Comment != nil {
		if err := c.Comment.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitColumn(c)
}

type ScalarTypeExpr struct {
	Name *Ident
}

func (s *ScalarTypeExpr) Pos() Pos {
	return s.Name.NamePos
}

func (s *ScalarTypeExpr) End() Pos {
	return s.Name.NameEnd
}

func (s *ScalarTypeExpr) String(level int) string {
	return s.Name.String(level + 1)
}

func (s *ScalarTypeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Name.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitScalarTypeExpr(s)
}

type PropertyTypeExpr struct {
	Name *Ident
}

func (c *PropertyTypeExpr) Pos() Pos {
	return c.Name.NamePos
}

func (c *PropertyTypeExpr) End() Pos {
	return c.Name.NameEnd
}

func (c *PropertyTypeExpr) String(level int) string {
	return c.Name.String(level + 1)
}

func (c *PropertyTypeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitPropertyTypeExpr(c)
}

type TypeWithParamsExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Name          *Ident
	Params        []Literal
}

func (s *TypeWithParamsExpr) Pos() Pos {
	return s.Name.NamePos
}

func (s *TypeWithParamsExpr) End() Pos {
	return s.RightParenPos
}

func (s *TypeWithParamsExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(s.Name.String(level))
	builder.WriteByte('(')
	for i, size := range s.Params {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(size.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (s *TypeWithParamsExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Name.Accept(visitor); err != nil {
		return err
	}
	for _, param := range s.Params {
		if err := param.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTypeWithParamsExpr(s)
}

type ComplexTypeExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Name          *Ident
	Params        []Expr
}

func (c *ComplexTypeExpr) Pos() Pos {
	return c.Name.NamePos
}

func (c *ComplexTypeExpr) End() Pos {
	return c.RightParenPos
}

func (c *ComplexTypeExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(c.Name.String(level))
	builder.WriteByte('(')
	for i, param := range c.Params {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(param.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (c *ComplexTypeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	for _, param := range c.Params {
		if err := param.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitComplexTypeExpr(c)
}

type NestedTypeExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Name          *Ident
	Columns       []Expr
}

func (n *NestedTypeExpr) Pos() Pos {
	return n.Name.NamePos
}

func (n *NestedTypeExpr) End() Pos {
	return n.RightParenPos
}

func (n *NestedTypeExpr) String(level int) string {
	var builder strings.Builder
	// on the same level as the column type
	builder.WriteString(n.Name.String(level))
	builder.WriteByte('(')
	for i, column := range n.Columns {
		builder.WriteString(NewLine(level + 2))
		builder.WriteString(column.String(level))
		if i != len(n.Columns)-1 {
			builder.WriteByte(',')
		}
	}
	// right paren needs to be on the same level as the column
	builder.WriteByte(')')
	return builder.String()
}

func (n *NestedTypeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Name.Accept(visitor); err != nil {
		return err
	}
	for _, column := range n.Columns {
		if err := column.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitNestedTypeExpr(n)
}

type CompressionCodec struct {
	CodecPos      Pos
	RightParenPos Pos
	Name          *Ident
	Level         *NumberLiteral // compression level
}

func (c *CompressionCodec) Pos() Pos {
	return c.CodecPos
}

func (c *CompressionCodec) End() Pos {
	return c.RightParenPos
}

func (c *CompressionCodec) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CODEC(")
	builder.WriteString(c.Name.String(level))
	if c.Level != nil {
		builder.WriteByte('(')
		builder.WriteString(c.Level.String(level))
		builder.WriteByte(')')
	}
	builder.WriteByte(')')
	return builder.String()
}

func (c *CompressionCodec) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	if c.Level != nil {
		if err := c.Level.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCompressionCodec(c)
}

type Literal interface {
	Expr
}

type NumberLiteral struct {
	NumPos  Pos
	NumEnd  Pos
	Literal string
	Base    int
}

func (n *NumberLiteral) Pos() Pos {
	return n.NumPos
}

func (n *NumberLiteral) End() Pos {
	return n.NumEnd
}

func (n *NumberLiteral) String(int) string {
	return n.Literal
}

func (n *NumberLiteral) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	return visitor.VisitNumberLiteral(n)
}

type StringLiteral struct {
	LiteralPos Pos
	LiteralEnd Pos
	Literal    string
}

func (s *StringLiteral) Pos() Pos {
	return s.LiteralPos
}

func (s *StringLiteral) End() Pos {
	return s.LiteralEnd
}

func (s *StringLiteral) String(int) string {
	return "'" + s.Literal + "'"
}

func (s *StringLiteral) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	return visitor.VisitStringLiteral(s)
}

type RatioExpr struct {
	Numerator *NumberLiteral
	// numberLiteral (SLASH numberLiteral)?
	Denominator *NumberLiteral
}

func (r *RatioExpr) Pos() Pos {
	return r.Numerator.NumPos
}

func (r *RatioExpr) End() Pos {
	if r.Denominator != nil {
		return r.Denominator.NumEnd
	}
	return r.Numerator.NumEnd
}

func (r *RatioExpr) String(int) string {
	var builder strings.Builder
	builder.WriteString(r.Numerator.String(0))
	if r.Denominator != nil {
		builder.WriteString("/")
		builder.WriteString(r.Denominator.String(0))
	}
	return builder.String()
}

func (r *RatioExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(r)
	defer visitor.leave(r)
	if err := r.Numerator.Accept(visitor); err != nil {
		return err
	}
	if r.Denominator != nil {
		if err := r.Denominator.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitRatioExpr(r)
}

type EnumValueExpr struct {
	Name  *StringLiteral
	Value *NumberLiteral
}

func (e *EnumValueExpr) Pos() Pos {
	return e.Name.Pos()
}

func (e *EnumValueExpr) End() Pos {
	return e.Value.End()
}

func (e *EnumValueExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(e.Name.String(level))
	builder.WriteByte('=')
	builder.WriteString(e.Value.String(level))
	return builder.String()
}

func (e *EnumValueExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(e)
	defer visitor.leave(e)
	if err := e.Name.Accept(visitor); err != nil {
		return err
	}
	if err := e.Value.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitEnumValueExpr(e)
}

type EnumValueExprList struct {
	ListPos Pos
	ListEnd Pos
	Enums   []EnumValueExpr
}

func (e *EnumValueExprList) Pos() Pos {
	return e.ListPos
}

func (e *EnumValueExprList) End() Pos {
	return e.ListEnd
}

func (e *EnumValueExprList) String(level int) string {
	var builder strings.Builder
	for i, enum := range e.Enums {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(enum.String(level))
	}
	return builder.String()
}

func (e *EnumValueExprList) Accept(visitor ASTVisitor) error {
	visitor.enter(e)
	defer visitor.leave(e)
	for i := range e.Enums {
		if err := e.Enums[i].Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitEnumValueExprList(e)
}

type IntervalExpr struct {
	IntervalPos Pos
	Expr        Expr
	Unit        *Ident
}

func (i *IntervalExpr) Pos() Pos {
	return i.IntervalPos
}

func (i *IntervalExpr) End() Pos {
	return i.Unit.End()
}

func (i *IntervalExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("INTERVAL ")
	builder.WriteString(i.Expr.String(level))
	builder.WriteByte(' ')
	builder.WriteString(i.Unit.String(level))
	return builder.String()
}

func (i *IntervalExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(i)
	defer visitor.leave(i)
	if err := i.Expr.Accept(visitor); err != nil {
		return err
	}
	if err := i.Unit.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitIntervalExpr(i)
}

type EngineExpr struct {
	EnginePos        Pos
	EngineEnd        Pos
	Name             string
	Params           *ParamExprList
	PrimaryKey       *PrimaryKeyExpr
	PartitionBy      *PartitionByExpr
	SampleBy         *SampleByExpr
	TTLExprList      *TTLExprList
	SettingsExprList *SettingsExprList
	OrderByListExpr  *OrderByListExpr
}

func (e *EngineExpr) Pos() Pos {
	return e.EnginePos
}

func (e *EngineExpr) End() Pos {
	return e.EngineEnd
}

func (e *EngineExpr) String(level int) string {
	// align with the engine level
	var builder strings.Builder
	builder.WriteString(NewLine(level))
	builder.WriteString("ENGINE = ")
	builder.WriteString(e.Name)
	if e.Params != nil {
		builder.WriteString(e.Params.String(level))
	}
	if e.PrimaryKey != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.PrimaryKey.String(level + 1))
	}
	if e.PartitionBy != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.PartitionBy.String(level + 1))
	}
	if e.SampleBy != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.SampleBy.String(level + 1))
	}
	if e.TTLExprList != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.TTLExprList.String(level + 1))
	}
	if e.SettingsExprList != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.SettingsExprList.String(level + 1))
	}
	if e.OrderByListExpr != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(e.OrderByListExpr.String(level + 1))
	}
	return builder.String()
}

func (e *EngineExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(e)
	defer visitor.leave(e)
	if e.Params != nil {
		if err := e.Params.Accept(visitor); err != nil {
			return err
		}
	}
	if e.PrimaryKey != nil {
		if err := e.PrimaryKey.Accept(visitor); err != nil {
			return err
		}
	}
	if e.PartitionBy != nil {
		if err := e.PartitionBy.Accept(visitor); err != nil {
			return err
		}
	}
	if e.SampleBy != nil {
		if err := e.SampleBy.Accept(visitor); err != nil {
			return err
		}
	}
	if e.TTLExprList != nil {
		if err := e.TTLExprList.Accept(visitor); err != nil {
			return err
		}
	}
	if e.SettingsExprList != nil {
		if err := e.SettingsExprList.Accept(visitor); err != nil {
			return err
		}
	}
	if e.OrderByListExpr != nil {
		if err := e.OrderByListExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitEngineExpr(e)
}

type ColumnTypeExpr struct {
	Name *Ident
}

func (c *ColumnTypeExpr) Pos() Pos {
	return c.Name.NamePos
}

func (c *ColumnTypeExpr) End() Pos {
	return c.Name.NameEnd
}

func (c *ColumnTypeExpr) String(level int) string {
	return c.Name.String(level)
}

func (c *ColumnTypeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitColumnTypeExpr(c)
}

type ColumnArgList struct {
	Distinct      bool
	LeftParenPos  Pos
	RightParenPos Pos
	Items         []Expr
}

func (c *ColumnArgList) Pos() Pos {
	return c.LeftParenPos
}

func (c *ColumnArgList) End() Pos {
	return c.RightParenPos
}

func (c *ColumnArgList) String(level int) string {
	var builder strings.Builder
	builder.WriteByte('(')
	for i, item := range c.Items {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(item.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (c *ColumnArgList) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	for _, item := range c.Items {
		if err := item.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitColumnArgList(c)
}

type ColumnExprList struct {
	ListPos     Pos
	ListEnd     Pos
	HasDistinct bool
	Items       []Expr
}

func (c *ColumnExprList) Pos() Pos {
	return c.ListPos
}

func (c *ColumnExprList) End() Pos {
	return c.ListEnd
}

func (c *ColumnExprList) String(level int) string {
	var builder strings.Builder
	if c.HasDistinct {
		builder.WriteString("DISTINCT ")
	}
	for i, item := range c.Items {
		builder.WriteString(item.String(level))
		if i != len(c.Items)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

func (c *ColumnExprList) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	for _, item := range c.Items {
		if err := item.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitColumnExprList(c)
}

type WhenExpr struct {
	WhenPos Pos
	ThenPos Pos
	When    Expr
	Then    Expr
	ElsePos Pos
	Else    Expr
}

func (w *WhenExpr) Pos() Pos {
	return w.WhenPos
}

func (w *WhenExpr) End() Pos {
	if w.Else != nil {
		return w.Else.End()
	}
	return w.Then.End()
}

func (w *WhenExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("WHEN ")
	builder.WriteString(NewLine(level + 1))
	builder.WriteString(w.When.String(level))
	builder.WriteString(NewLine(level + 1))
	builder.WriteString(" THEN ")
	builder.WriteString(w.Then.String(level))
	if w.Else != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(" ELSE ")
		builder.WriteString(w.Else.String(level))
	}
	return builder.String()
}

func (w *WhenExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if err := w.When.Accept(visitor); err != nil {
		return err
	}
	if err := w.Then.Accept(visitor); err != nil {
		return err
	}
	if w.Else != nil {
		if err := w.Else.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitWhenExpr(w)
}

type CaseExpr struct {
	CasePos Pos
	EndPos  Pos
	Expr    Expr
	Whens   []*WhenExpr
	ElsePos Pos
	Else    Expr
}

func (c *CaseExpr) Pos() Pos {
	return c.CasePos
}

func (c *CaseExpr) End() Pos {
	return c.EndPos
}

func (c *CaseExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CASE ")
	builder.WriteString(NewLine(level))
	builder.WriteString(c.Expr.String(level))
	for _, when := range c.Whens {
		builder.WriteString(NewLine(level))
		builder.WriteString(when.String(level))
	}
	if c.Else != nil {
		builder.WriteString("ELSE ")
		builder.WriteString(NewLine(level))
		builder.WriteString(c.Else.String(level))
	}
	builder.WriteString(NewLine(level))
	builder.WriteString("END")
	return builder.String()
}

func (c *CaseExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Expr.Accept(visitor); err != nil {
		return err
	}
	for _, when := range c.Whens {
		if err := when.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Else != nil {
		if err := c.Else.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCaseExpr(c)
}

type CastExpr struct {
	CastPos   Pos
	Expr      Expr
	Separator string
	AsPos     Pos
	AsType    Expr
}

func (c *CastExpr) Pos() Pos {
	return c.CastPos
}

func (c *CastExpr) End() Pos {
	return c.AsType.End()
}

func (c *CastExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CAST(")
	builder.WriteString(c.Expr.String(level))
	if c.Separator == "," {
		builder.WriteString(", ")
	} else {
		builder.WriteString(" AS ")
	}
	builder.WriteString(c.AsType.String(level))
	builder.WriteByte(')')
	return builder.String()
}

func (c *CastExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Expr.Accept(visitor); err != nil {
		return err
	}
	if err := c.AsType.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitCastExpr(c)
}

type WithExpr struct {
	WithPos Pos
	EndPos  Pos
	CTEs    []*CTEExpr
}

func (w *WithExpr) Pos() Pos {
	return w.WithPos
}

func (w *WithExpr) End() Pos {
	return w.EndPos
}

func (w *WithExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("WITH ")
	for i, cte := range w.CTEs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(cte.String(level + 1))
	}
	return builder.String()
}

func (w *WithExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	for _, cte := range w.CTEs {
		if err := cte.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitWithExpr(w)
}

type TopExpr struct {
	TopPos   Pos
	TopEnd   Pos
	Number   *NumberLiteral
	WithTies bool
}

func (t *TopExpr) Pos() Pos {
	return t.TopPos
}

func (t *TopExpr) End() Pos {
	return t.TopEnd
}

func (t *TopExpr) String(int) string {
	var builder strings.Builder
	builder.WriteString("TOP ")
	builder.WriteString(t.Number.Literal)
	if t.WithTies {
		return "WITH TIES"
	}
	return builder.String()
}

func (t *TopExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.Number.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitTopExpr(t)
}

type CreateLiveView struct {
	CreatePos    Pos
	StatementEnd Pos
	Name         *TableIdentifier
	IfNotExists  bool
	UUID         *UUID
	OnCluster    *OnClusterExpr
	Destination  *DestinationExpr
	TableSchema  *TableSchemaExpr
	WithTimeout  *WithTimeoutExpr
	SubQuery     *SubQueryExpr
}

func (c *CreateLiveView) Type() string {
	return "LIVE_VIEW"
}

func (c *CreateLiveView) Pos() Pos {
	return c.CreatePos
}

func (c *CreateLiveView) End() Pos {
	return c.StatementEnd
}

func (c *CreateLiveView) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CREATE LIVE VIEW ")
	if c.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(c.Name.String(level))

	if c.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.OnCluster.String(level))
	}

	if c.WithTimeout != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.WithTimeout.String(level))
	}

	if c.Destination != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.Destination.String(level))
	}

	if c.TableSchema != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(c.TableSchema.String(level))
	}

	if c.SubQuery != nil {
		builder.WriteString(c.SubQuery.String(level))
	}

	return builder.String()
}

func (c *CreateLiveView) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Name.Accept(visitor); err != nil {
		return err
	}
	if c.UUID != nil {
		if err := c.UUID.Accept(visitor); err != nil {
			return err
		}
	}
	if c.OnCluster != nil {
		if err := c.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if c.Destination != nil {
		if err := c.Destination.Accept(visitor); err != nil {
			return err
		}
	}
	if c.TableSchema != nil {
		if err := c.TableSchema.Accept(visitor); err != nil {
			return err
		}
	}
	if c.WithTimeout != nil {
		if err := c.WithTimeout.Accept(visitor); err != nil {
			return err
		}
	}
	if c.SubQuery != nil {
		if err := c.SubQuery.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCreateLiveView(c)
}

type WithTimeoutExpr struct {
	WithTimeoutPos Pos
	Expr           Expr
	Number         *NumberLiteral
}

func (w *WithTimeoutExpr) Pos() Pos {
	return w.WithTimeoutPos
}

func (w *WithTimeoutExpr) End() Pos {
	return w.Number.End()
}

func (w *WithTimeoutExpr) String(int) string {
	var builder strings.Builder
	builder.WriteString("WITH TIMEOUT ")
	builder.WriteString(w.Number.String(0))
	return builder.String()
}

func (w *WithTimeoutExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if err := w.Number.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWithTimeoutExpr(w)
}

type TableExpr struct {
	TablePos Pos
	TableEnd Pos
	Alias    *AliasExpr
	Expr     Expr
	HasFinal bool
}

func (t *TableExpr) Pos() Pos {
	return t.TablePos
}

func (t *TableExpr) End() Pos {
	return t.TableEnd
}

func (t *TableExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(t.Expr.String(level + 1))
	if t.Alias != nil {
		builder.WriteByte(' ')
		builder.WriteString(t.Alias.String(level + 1))
	}
	if t.HasFinal {
		builder.WriteString(" FINAL")
	}
	return builder.String()
}

func (t *TableExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.Expr.Accept(visitor); err != nil {
		return err
	}
	if t.Alias != nil {
		if err := t.Alias.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTableExpr(t)
}

type OnExpr struct {
	OnPos Pos
	On    *ColumnExprList
}

func (o *OnExpr) Pos() Pos {
	return o.OnPos
}

func (o *OnExpr) End() Pos {
	return o.On.End()
}

func (o *OnExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("ON ")
	builder.WriteString(o.On.String(level))
	return builder.String()
}

func (o *OnExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	if err := o.On.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitOnExpr(o)
}

type UsingExpr struct {
	UsingPos Pos
	Using    *ColumnExprList
}

func (u *UsingExpr) Pos() Pos {
	return u.UsingPos
}

func (u *UsingExpr) End() Pos {
	return u.Using.End()
}

func (u *UsingExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("USING ")
	builder.WriteString(u.Using.String(level))
	return builder.String()
}

func (u *UsingExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(u)
	defer visitor.leave(u)
	if err := u.Using.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitUsingExpr(u)
}

type JoinExpr struct {
	JoinPos     Pos
	Left        Expr
	Right       Expr
	Modifiers   []string
	Constraints Expr
}

func (j *JoinExpr) Pos() Pos {
	return j.JoinPos
}

func (j *JoinExpr) End() Pos {
	return j.Left.End()
}

func buildJoinString(builder *strings.Builder, expr Expr, level int) {
	joinExpr, ok := expr.(*JoinExpr)
	if !ok {
		builder.WriteString(",")
		builder.WriteString(expr.String(level))
		return
	}

	if len(joinExpr.Modifiers) == 0 {
		builder.WriteString(",")
	} else {
		builder.WriteString(NewLine(level))
		builder.WriteString(strings.Join(joinExpr.Modifiers, " "))
		builder.WriteByte(' ')
	}
	builder.WriteString(joinExpr.Left.String(level))
	if joinExpr.Constraints != nil {
		builder.WriteByte(' ')
		builder.WriteString(joinExpr.Constraints.String(level))
	}
	if joinExpr.Right != nil {
		buildJoinString(builder, joinExpr.Right, level)
	}
}

func (j *JoinExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(j.Left.String(level))
	if j.Right != nil {
		buildJoinString(&builder, j.Right, level)
	}
	return builder.String()
}

func (j *JoinExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(j)
	defer visitor.leave(j)
	if err := j.Left.Accept(visitor); err != nil {
		return err
	}
	if j.Right != nil {
		if err := j.Right.Accept(visitor); err != nil {
			return err
		}
	}
	if j.Constraints != nil {
		if err := j.Constraints.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitJoinExpr(j)
}

type JoinConstraintExpr struct {
	ConstraintPos Pos
	On            *ColumnExprList
	Using         *ColumnExprList
}

func (j *JoinConstraintExpr) Pos() Pos {
	return j.ConstraintPos
}

func (j *JoinConstraintExpr) End() Pos {
	if j.On != nil {
		return j.On.End()
	}
	return j.Using.End()
}

func (j *JoinConstraintExpr) String(level int) string {
	var builder strings.Builder
	if j.On != nil {
		builder.WriteString("ON ")
		builder.WriteString(j.On.String(level))
	} else {
		builder.WriteString("USING ")
		builder.WriteString(j.Using.String(level))
	}
	return builder.String()
}

func (j *JoinConstraintExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(j)
	defer visitor.leave(j)
	if j.On != nil {
		if err := j.On.Accept(visitor); err != nil {
			return err
		}
	}
	if j.Using != nil {
		if err := j.Using.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitJoinConstraintExpr(j)
}

type FromExpr struct {
	FromPos Pos
	Expr    Expr
}

func (f *FromExpr) Pos() Pos {
	return f.FromPos
}

func (f *FromExpr) End() Pos {
	return f.Expr.End()
}

func (f *FromExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("FROM")

	isSubquery := false
	joinTableExpr, ok := f.Expr.(*JoinTableExpr)
	if ok && joinTableExpr.Table != nil {
		_, ok := joinTableExpr.Table.Expr.(*SelectQuery)
		if ok {
			isSubquery = true
		}
	}
	if isSubquery {
		builder.WriteString(" (")
	}
	builder.WriteString(NewLine(level + 1))
	builder.WriteString(f.Expr.String(level + 1))
	if isSubquery {
		builder.WriteString(")")
	}
	return builder.String()
}

func (f *FromExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitFromExpr(f)
}

type IsNullExpr struct {
	IsPos Pos
	Expr  Expr
}

func (n *IsNullExpr) Pos() Pos {
	return n.IsPos
}

func (n *IsNullExpr) End() Pos {
	return n.Expr.End()
}

func (n *IsNullExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(n.Expr.String(level))
	builder.WriteString(" IS NULL")
	return builder.String()
}

func (n *IsNullExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitIsNullExpr(n)
}

type IsNotNullExpr struct {
	IsPos Pos
	Expr  Expr
}

func (n *IsNotNullExpr) Pos() Pos {
	return n.Expr.Pos()
}

func (n *IsNotNullExpr) End() Pos {
	return n.Expr.End()
}

func (n *IsNotNullExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(n.Expr.String(level))
	builder.WriteString(" IS NOT NULL")
	return builder.String()
}

func (n *IsNotNullExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitIsNotNullExpr(n)
}

type AliasExpr struct {
	Expr     Expr
	AliasPos Pos
	Alias    Expr
}

func (a *AliasExpr) Pos() Pos {
	return a.AliasPos
}

func (a *AliasExpr) End() Pos {
	return a.Alias.End()
}

func (a *AliasExpr) String(level int) string {
	var builder strings.Builder
	if _, isSelect := a.Expr.(*SelectQuery); isSelect {
		builder.WriteByte('(')
		builder.WriteString(a.Expr.String(level))
		builder.WriteByte(')')
	} else {
		builder.WriteString(a.Expr.String(level))
	}
	builder.WriteString(" AS ")
	builder.WriteString(a.Alias.String(level))
	return builder.String()
}

func (a *AliasExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Expr.Accept(visitor); err != nil {
		return err
	}
	if err := a.Alias.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitAliasExpr(a)
}

type WhereExpr struct {
	WherePos Pos
	Expr     Expr
}

func (w *WhereExpr) Pos() Pos {
	return w.WherePos
}

func (w *WhereExpr) End() Pos {
	return w.Expr.End()
}

func (w *WhereExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("WHERE")
	builder.WriteString(NewLine(level + 1))
	builder.WriteString(w.Expr.String(level))
	return builder.String()
}

func (w *WhereExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if err := w.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWhereExpr(w)
}

type PrewhereExpr struct {
	PrewherePos Pos
	Expr        Expr
}

func (w *PrewhereExpr) Pos() Pos {
	return w.PrewherePos
}

func (w *PrewhereExpr) End() Pos {
	return w.Expr.End()
}

func (w *PrewhereExpr) String(level int) string {
	return "PREWHERE " + w.Expr.String(level+1)
}

func (w *PrewhereExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if err := w.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitPrewhereExpr(w)
}

type GroupByExpr struct {
	GroupByPos    Pos
	AggregateType string
	Expr          Expr
	WithCube      bool
	WithRollup    bool
	WithTotals    bool
}

func (g *GroupByExpr) Pos() Pos {
	return g.GroupByPos
}

func (g *GroupByExpr) End() Pos {
	return g.Expr.End()
}

func (g *GroupByExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("GROUP BY ")
	if g.AggregateType != "" {
		builder.WriteString(g.AggregateType)
		builder.WriteByte('(')
	}
	builder.WriteString(g.Expr.String(level))
	if g.AggregateType != "" {
		builder.WriteByte(')')
	}
	if g.WithCube {
		builder.WriteString(" WITH CUBE")
	}
	if g.WithRollup {
		builder.WriteString(" WITH ROLLUP")
	}
	if g.WithTotals {
		builder.WriteString(" WITH TOTALS")
	}
	return builder.String()
}

func (g *GroupByExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(g)
	defer visitor.leave(g)
	if err := g.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitGroupByExpr(g)
}

type HavingExpr struct {
	HavingPos Pos
	Expr      Expr
}

func (h *HavingExpr) Pos() Pos {
	return h.HavingPos
}

func (h *HavingExpr) End() Pos {
	return h.Expr.End()
}

func (h *HavingExpr) String(level int) string {
	return "HAVING " + h.Expr.String(level)
}

func (h *HavingExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(h)
	defer visitor.leave(h)
	if err := h.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitHavingExpr(h)
}

type LimitExpr struct {
	LimitPos Pos
	Limit    Expr
	Offset   Expr
}

func (l *LimitExpr) Pos() Pos {
	return l.LimitPos
}

func (l *LimitExpr) End() Pos {
	if l.Offset != nil {
		return l.Offset.End()
	}
	return l.Limit.End()
}

func (l *LimitExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("LIMIT ")
	builder.WriteString(l.Limit.String(level))
	if l.Offset != nil {
		builder.WriteString(" OFFSET ")
		builder.WriteString(l.Offset.String(level))
	}
	return builder.String()
}

func (l *LimitExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(l)
	defer visitor.leave(l)
	if err := l.Limit.Accept(visitor); err != nil {
		return err
	}
	if l.Offset != nil {
		if err := l.Offset.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitLimitExpr(l)
}

type LimitByExpr struct {
	Limit  *LimitExpr
	ByExpr *ColumnExprList
}

func (l *LimitByExpr) Pos() Pos {
	return l.Limit.Pos()
}

func (l *LimitByExpr) End() Pos {
	if l.ByExpr != nil {
		return l.ByExpr.End()
	}
	if l.Limit != nil {
		return l.Limit.End()
	}
	return l.Limit.End()
}

func (l *LimitByExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("LIMIT ")
	builder.WriteString(l.Limit.String(level))
	if l.Limit != nil {
		builder.WriteString(" OFFSET ")
		builder.WriteString(l.Limit.String(level))
	}
	if l.ByExpr != nil {
		builder.WriteString(" BY ")
		builder.WriteString(l.ByExpr.String(level))
	}
	return builder.String()
}

func (l *LimitByExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(l)
	defer visitor.leave(l)
	if l.Limit != nil {
		if err := l.Limit.Accept(visitor); err != nil {
			return err
		}
	}
	if l.ByExpr != nil {
		if err := l.ByExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitLimitByExpr(l)
}

type WindowConditionExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	PartitionBy   *PartitionByExpr
	OrderBy       *OrderByListExpr
	Frame         *WindowFrameExpr
}

func (w *WindowConditionExpr) Pos() Pos {
	return w.LeftParenPos
}

func (w *WindowConditionExpr) End() Pos {
	return w.RightParenPos
}

func (w *WindowConditionExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteByte('(')
	if w.PartitionBy != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(w.PartitionBy.String(level))
	}
	if w.OrderBy != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(w.OrderBy.String(level))
	}
	if w.Frame != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(w.Frame.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (w *WindowConditionExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if w.PartitionBy != nil {
		if err := w.PartitionBy.Accept(visitor); err != nil {
			return err
		}
	}
	if w.OrderBy != nil {
		if err := w.OrderBy.Accept(visitor); err != nil {
			return err
		}
	}
	if w.Frame != nil {
		if err := w.Frame.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitWindowConditionExpr(w)
}

type WindowExpr struct {
	*WindowConditionExpr

	WindowPos Pos
	Name      *Ident
	AsPos     Pos
}

func (w *WindowExpr) Pos() Pos {
	return w.WindowPos
}

func (w *WindowExpr) End() Pos {
	return w.WindowConditionExpr.End()
}

func (w *WindowExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("WINDOW ")
	builder.WriteString(w.Name.String(level))
	builder.WriteString(" ")
	builder.WriteString(w.WindowConditionExpr.String(level))
	return builder.String()
}

func (w *WindowExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(w)
	defer visitor.leave(w)
	if w.WindowConditionExpr != nil {
		if err := w.WindowConditionExpr.Accept(visitor); err != nil {
			return err
		}
	}
	if w.Name != nil {
		if err := w.Name.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitWindowExpr(w)
}

type WindowFrameExpr struct {
	FramePos Pos
	Type     string
	Extend   Expr
}

func (f *WindowFrameExpr) Pos() Pos {
	return f.FramePos
}

func (f *WindowFrameExpr) End() Pos {
	return f.Extend.End()
}

func (f *WindowFrameExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(f.Type)
	builder.WriteString(" ")
	builder.WriteString(f.Extend.String(level))
	return builder.String()
}

func (f *WindowFrameExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Extend.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWindowFrameExpr(f)
}

type WindowFrameExtendExpr struct {
	Expr Expr
}

func (f *WindowFrameExtendExpr) Pos() Pos {
	return f.Expr.Pos()
}

func (f *WindowFrameExtendExpr) End() Pos {
	return f.Expr.End()
}

func (f *WindowFrameExtendExpr) String(int) string {
	return f.Expr.String(0)
}

func (f *WindowFrameExtendExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWindowFrameExtendExpr(f)
}

type WindowFrameRangeExpr struct {
	BetweenPos  Pos
	BetweenExpr Expr
	AndPos      Pos
	AndExpr     Expr
}

func (f *WindowFrameRangeExpr) Pos() Pos {
	return f.BetweenPos
}

func (f *WindowFrameRangeExpr) End() Pos {
	return f.AndExpr.End()
}

func (f *WindowFrameRangeExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("BETWEEN ")
	builder.WriteString(f.BetweenExpr.String(level))
	builder.WriteString(" AND ")
	builder.WriteString(f.AndExpr.String(level))
	return builder.String()
}

func (f *WindowFrameRangeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.BetweenExpr.Accept(visitor); err != nil {
		return err
	}
	if err := f.AndExpr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWindowFrameRangeExpr(f)
}

type WindowFrameCurrentRow struct {
	CurrentPos Pos
	RowEnd     Pos
}

func (f *WindowFrameCurrentRow) Pos() Pos {
	return f.CurrentPos
}

func (f *WindowFrameCurrentRow) End() Pos {
	return f.RowEnd
}

func (f *WindowFrameCurrentRow) String(int) string {
	return "CURRENT ROW"
}

func (f *WindowFrameCurrentRow) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	return visitor.VisitWindowFrameCurrentRow(f)
}

type WindowFrameUnbounded struct {
	UnboundedPos Pos
	UnboundedEnd Pos
	Direction    string
}

func (f *WindowFrameUnbounded) Pos() Pos {
	return f.UnboundedPos
}

func (f *WindowFrameUnbounded) End() Pos {
	return f.UnboundedEnd
}

func (f *WindowFrameUnbounded) String(int) string {
	return f.Direction + " UNBOUNDED"
}

func (f *WindowFrameUnbounded) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	return visitor.VisitWindowFrameUnbounded(f)
}

type WindowFrameNumber struct {
	Number       *NumberLiteral
	UnboundedEnd Pos
	Direction    string
}

func (f *WindowFrameNumber) Pos() Pos {
	return f.Number.Pos()
}

func (f *WindowFrameNumber) End() Pos {
	return f.UnboundedEnd
}

func (f *WindowFrameNumber) String(level int) string {
	var builder strings.Builder
	builder.WriteString(f.Number.String(level))
	builder.WriteByte(' ')
	builder.WriteString(f.Direction)
	return builder.String()
}

func (f *WindowFrameNumber) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Number.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitWindowFrameNumber(f)
}

type ArrayJoinExpr struct {
	ArrayPos Pos
	Type     string
	Expr     Expr
}

func (a *ArrayJoinExpr) Pos() Pos {
	return a.ArrayPos
}

func (a *ArrayJoinExpr) End() Pos {
	return a.Expr.End()
}

func (a *ArrayJoinExpr) String(level int) string {
	return a.Type + " ARRAY JOIN " + a.Expr.String(level)
}

func (a *ArrayJoinExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(a)
	defer visitor.leave(a)
	if err := a.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitArrayJoinExpr(a)
}

type SelectQuery struct {
	SelectPos     Pos
	StatementEnd  Pos
	With          *WithExpr
	Top           *TopExpr
	SelectColumns *ColumnExprList
	From          *FromExpr
	ArrayJoin     *ArrayJoinExpr
	Window        *WindowExpr
	Prewhere      *PrewhereExpr
	Where         *WhereExpr
	GroupBy       *GroupByExpr
	WithTotal     bool
	Having        *HavingExpr
	OrderBy       *OrderByListExpr
	LimitBy       *LimitByExpr
	Limit         *LimitExpr
	Settings      *SettingsExprList
	UnionAll      *SelectQuery
	UnionDistinct *SelectQuery
	Except        *SelectQuery
}

func (s *SelectQuery) Pos() Pos {
	return s.SelectPos
}

func (s *SelectQuery) End() Pos {
	return s.StatementEnd
}

func (s *SelectQuery) String(level int) string { // nolint: funlen
	var builder strings.Builder
	if s.With != nil {
		builder.WriteString("WITH")
		for i, cte := range s.With.CTEs {
			builder.WriteString(NewLine(level + 1))
			builder.WriteString(cte.String(level))
			if i != len(s.With.CTEs)-1 {
				builder.WriteByte(',')
			}
		}
	}
	builder.WriteString(NewLine(level))
	builder.WriteString("SELECT ")
	if s.Top != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(s.Top.String(level))
		builder.WriteString(" ")
	}
	if s.SelectColumns.HasDistinct {
		builder.WriteString("DISTINCT ")
	}
	columns := s.SelectColumns.Items
	for i, column := range columns {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(column.String(level))
		if i != len(columns)-1 {
			builder.WriteByte(',')
		}
	}
	if s.From != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.From.String(level))
	}
	if s.ArrayJoin != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.ArrayJoin.String(level))
	}
	if s.Window != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Window.String(level))
	}
	if s.Prewhere != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Prewhere.String(level))
	}
	if s.Where != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Where.String(level))
	}
	if s.GroupBy != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.GroupBy.String(level))
	}
	if s.Having != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Having.String(level))
	}
	if s.OrderBy != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.OrderBy.String(level))
	}
	if s.LimitBy != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.LimitBy.String(level))
	}
	if s.Limit != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Limit.String(level))
	}
	if s.Settings != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(s.Settings.String(level))
	}
	if s.UnionAll != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(" UNION ALL ")
		builder.WriteString(s.UnionAll.String(level))
	} else if s.UnionDistinct != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(" UNION DISTINCT ")
		builder.WriteString(s.UnionDistinct.String(level))
	} else if s.Except != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(" EXCEPT ")
		builder.WriteString(s.Except.String(level))
	}
	return builder.String()
}

func (s *SelectQuery) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if s.With != nil {
		if err := s.With.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Top != nil {
		if err := s.Top.Accept(visitor); err != nil {
			return err
		}
	}
	if s.SelectColumns != nil {
		if err := s.SelectColumns.Accept(visitor); err != nil {
			return err
		}
	}
	if s.From != nil {
		if err := s.From.Accept(visitor); err != nil {
			return err
		}
	}
	if s.ArrayJoin != nil {
		if err := s.ArrayJoin.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Window != nil {
		if err := s.Window.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Prewhere != nil {
		if err := s.Prewhere.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Where != nil {
		if err := s.Where.Accept(visitor); err != nil {
			return err
		}
	}
	if s.GroupBy != nil {
		if err := s.GroupBy.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Having != nil {
		if err := s.Having.Accept(visitor); err != nil {
			return err
		}
	}
	if s.OrderBy != nil {
		if err := s.OrderBy.Accept(visitor); err != nil {
			return err
		}
	}
	if s.LimitBy != nil {
		if err := s.LimitBy.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Limit != nil {
		if err := s.Limit.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Settings != nil {
		if err := s.Settings.Accept(visitor); err != nil {
			return err
		}
	}
	if s.UnionAll != nil {
		if err := s.UnionAll.Accept(visitor); err != nil {
			return err
		}
	}
	if s.UnionDistinct != nil {
		if err := s.UnionDistinct.Accept(visitor); err != nil {
			return err
		}
	}
	if s.Except != nil {
		if err := s.Except.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSelectQuery(s)
}

type SubQueryExpr struct {
	AsPos  Pos
	Select *SelectQuery
}

func (s *SubQueryExpr) Pos() Pos {
	return s.AsPos
}

func (s *SubQueryExpr) End() Pos {
	return s.Select.End()
}

func (s *SubQueryExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(" AS (")
	builder.WriteString(s.Select.String(level + 1))
	builder.WriteString(NewLine(level))
	builder.WriteString(")")
	return builder.String()
}

func (s *SubQueryExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if s.Select != nil {
		if err := s.Select.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSubQueryExpr(s)
}

type NotExpr struct {
	NotPos Pos
	Expr   Expr
}

func (n *NotExpr) Pos() Pos {
	return n.NotPos
}

func (n *NotExpr) End() Pos {
	return n.Expr.End()
}

func (n *NotExpr) String(level int) string {
	return "NOT " + n.Expr.String(level+1)
}

func (n *NotExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitNotExpr(n)
}

type NegateExpr struct {
	NegatePos Pos
	Expr      Expr
}

func (n *NegateExpr) Pos() Pos {
	return n.NegatePos
}

func (n *NegateExpr) End() Pos {
	return n.Expr.End()
}

func (n *NegateExpr) String(level int) string {
	return "-" + n.Expr.String(level+1)
}

func (n *NegateExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitNegateExpr(n)
}

type GlobalInExpr struct {
	GlobalPos Pos
	Expr      Expr
}

func (g *GlobalInExpr) Pos() Pos {
	return g.GlobalPos
}

func (g *GlobalInExpr) End() Pos {
	return g.Expr.End()
}

func (g *GlobalInExpr) String(level int) string {
	return "GLOBAL " + g.Expr.String(level+1)
}

func (g *GlobalInExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(g)
	defer visitor.leave(g)
	if err := g.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitGlobalInExpr(g)
}

type ExtractExpr struct {
	ExtractPos Pos
	Interval   *Ident
	FromPos    Pos
	FromExpr   Expr
}

func (e *ExtractExpr) Pos() Pos {
	return e.ExtractPos
}

func (e *ExtractExpr) End() Pos {
	return e.FromExpr.End()
}

func (e *ExtractExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("EXTRACT(")
	builder.WriteString(e.Interval.String(level))
	builder.WriteString(" FROM ")
	builder.WriteString(e.FromExpr.String(level))
	builder.WriteByte(')')
	return builder.String()
}

func (e *ExtractExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(e)
	defer visitor.leave(e)
	if err := e.FromExpr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitExtractExpr(e)
}

type DropDatabase struct {
	DropPos      Pos
	StatementEnd Pos
	Name         *Ident
	IfExists     bool
	OnCluster    *OnClusterExpr
}

func (d *DropDatabase) Pos() Pos {
	return d.DropPos
}

func (d *DropDatabase) End() Pos {
	return d.StatementEnd
}

func (d *DropDatabase) Type() string {
	return "DATABASE"
}

func (d *DropDatabase) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP DATABASE ")
	if d.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(d.Name.String(level))
	if d.OnCluster != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(d.OnCluster.String(level))
	}
	return builder.String()
}

func (d *DropDatabase) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if err := d.Name.Accept(visitor); err != nil {
		return err
	}
	if d.OnCluster != nil {
		if err := d.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitDropDatabase(d)
}

type DropStmt struct {
	DropPos      Pos
	StatementEnd Pos

	DropTarget  string
	Name        *TableIdentifier
	IfExists    bool
	OnCluster   *OnClusterExpr
	IsTemporary bool
	Modifier    string
}

func (d *DropStmt) Pos() Pos {
	return d.DropPos
}

func (d *DropStmt) End() Pos {
	return d.StatementEnd
}

func (d *DropStmt) Type() string {
	return "DROP " + d.DropTarget
}

func (d *DropStmt) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP ")
	if d.IsTemporary {
		builder.WriteString("TEMPORARY ")
	}
	builder.WriteString(d.DropTarget + " ")
	if d.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(d.Name.String(level))
	if d.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(d.OnCluster.String(level))
	}
	if len(d.Modifier) != 0 {
		builder.WriteString(" " + d.Modifier)
	}
	return builder.String()
}

func (d *DropStmt) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if err := d.Name.Accept(visitor); err != nil {
		return err
	}
	if d.OnCluster != nil {
		if err := d.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitDropStmt(d)

}

type DropUserOrRole struct {
	DropPos      Pos
	Target       string
	StatementEnd Pos
	Names        []*RoleName
	IfExists     bool
	Modifier     string
	From         *Ident
}

func (d *DropUserOrRole) Pos() Pos {
	return d.DropPos
}

func (d *DropUserOrRole) End() Pos {
	return d.StatementEnd
}

func (d *DropUserOrRole) Type() string {
	return d.Target
}

func (d *DropUserOrRole) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DROP " + d.Target + " ")
	if d.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	for i, name := range d.Names {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(name.String(level))
	}
	if len(d.Modifier) != 0 {
		builder.WriteString(" " + d.Modifier)
	}
	if d.From != nil {
		builder.WriteString(" FROM ")
		builder.WriteString(d.From.String(level))
	}
	return builder.String()
}

func (d *DropUserOrRole) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	for _, name := range d.Names {
		if err := name.Accept(visitor); err != nil {
			return err
		}
	}
	if d.From != nil {
		if err := d.From.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitDropUserOrRole(d)
}

type UseExpr struct {
	UsePos       Pos
	StatementEnd Pos
	Database     *Ident
}

func (u *UseExpr) Pos() Pos {
	return u.UsePos
}

func (u *UseExpr) End() Pos {
	return u.Database.End()
}

func (u *UseExpr) String(level int) string {
	return "USE " + u.Database.String(level+1)
}

func (u *UseExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(u)
	defer visitor.leave(u)
	if err := u.Database.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitUseExpr(u)
}

type CTEExpr struct {
	CTEPos Pos
	Expr   Expr
	Alias  Expr
}

func (c *CTEExpr) Pos() Pos {
	return c.CTEPos
}

func (c *CTEExpr) End() Pos {
	return c.Expr.End()
}

func (c *CTEExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(c.Expr.String(level))
	builder.WriteString(" AS ")
	if _, isSelect := c.Alias.(*SelectQuery); isSelect {
		builder.WriteByte('(')
		builder.WriteString(c.Alias.String(level + 2))
		builder.WriteByte(')')
	} else {
		builder.WriteString(c.Alias.String(level))
	}
	return builder.String()
}

func (c *CTEExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Expr.Accept(visitor); err != nil {
		return err
	}
	if err := c.Alias.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitCTEExpr(c)
}

type SetExpr struct {
	SetPos   Pos
	Settings *SettingsExprList
}

func (s *SetExpr) Pos() Pos {
	return s.SetPos
}

func (s *SetExpr) End() Pos {
	return s.Settings.End()
}

func (s *SetExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("SET ")
	for i, item := range s.Settings.Items {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(item.String(level))
	}
	return builder.String()
}

func (s *SetExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Settings.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitSetExpr(s)
}

type FormatExpr struct {
	FormatPos Pos
	Format    *Ident
}

func (f *FormatExpr) Pos() Pos {
	return f.FormatPos
}

func (f *FormatExpr) End() Pos {
	return f.Format.End()
}

func (f *FormatExpr) String(level int) string {
	return "FORMAT " + f.Format.String(level)
}

func (f *FormatExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(f)
	defer visitor.leave(f)
	if err := f.Format.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitFormatExpr(f)
}

type OptimizeExpr struct {
	OptimizePos  Pos
	StatementEnd Pos
	Table        *TableIdentifier
	OnCluster    *OnClusterExpr
	Partition    *PartitionExpr
	HasFinal     bool
	Deduplicate  *DeduplicateExpr
}

func (o *OptimizeExpr) Pos() Pos {
	return o.OptimizePos
}

func (o *OptimizeExpr) End() Pos {
	return o.StatementEnd
}

func (o *OptimizeExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("OPTIMIZE TABLE ")
	builder.WriteString(o.Table.String(level))
	if o.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(o.OnCluster.String(level))
	}
	if o.Partition != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(o.Partition.String(level))
	}
	if o.HasFinal {
		builder.WriteString(" FINAL")
	}
	if o.Deduplicate != nil {
		builder.WriteString(o.Deduplicate.String(level))
	}
	return builder.String()
}

func (o *OptimizeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(o)
	defer visitor.leave(o)
	if err := o.Table.Accept(visitor); err != nil {
		return err
	}
	if o.OnCluster != nil {
		if err := o.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if o.Partition != nil {
		if err := o.Partition.Accept(visitor); err != nil {
			return err
		}
	}
	if o.Deduplicate != nil {
		if err := o.Deduplicate.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitOptimizeExpr(o)
}

type DeduplicateExpr struct {
	DeduplicatePos Pos
	By             *ColumnExprList
	Except         *ColumnExprList
}

func (d *DeduplicateExpr) Pos() Pos {
	return d.DeduplicatePos
}

func (d *DeduplicateExpr) End() Pos {
	if d.By != nil {
		return d.By.End()
	} else if d.Except != nil {
		return d.Except.End()
	}
	return d.DeduplicatePos + Pos(len(KeywordDeduplicate))
}

func (d *DeduplicateExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(" DEDUPLICATE")
	if d.By != nil {
		builder.WriteString(" BY ")
		builder.WriteString(d.By.String(level))
	}
	if d.Except != nil {
		builder.WriteString(" EXCEPT ")
		builder.WriteString(d.Except.String(level))
	}
	return builder.String()
}

func (d *DeduplicateExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if d.By != nil {
		if err := d.By.Accept(visitor); err != nil {
			return err
		}
	}
	if d.Except != nil {
		if err := d.Except.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitDeduplicateExpr(d)
}

type SystemExpr struct {
	SystemPos Pos
	Expr      Expr
}

func (s *SystemExpr) Pos() Pos {
	return s.SystemPos
}

func (s *SystemExpr) End() Pos {
	return s.Expr.End()
}

func (s *SystemExpr) String(level int) string {
	return "SYSTEM " + s.Expr.String(level)
}

func (s *SystemExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitSystemExpr(s)
}

type SystemFlushExpr struct {
	FlushPos     Pos
	StatementEnd Pos
	Logs         bool
	Distributed  *TableIdentifier
}

func (s *SystemFlushExpr) Pos() Pos {
	return s.FlushPos
}

func (s *SystemFlushExpr) End() Pos {
	return s.StatementEnd
}

func (s *SystemFlushExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("FLUSH ")
	if s.Logs {
		builder.WriteString("LOGS")
	} else {
		builder.WriteString(s.Distributed.String(level))
	}
	return builder.String()
}

func (s *SystemFlushExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if s.Distributed != nil {
		if err := s.Distributed.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSystemFlushExpr(s)
}

type SystemReloadExpr struct {
	ReloadPos    Pos
	StatementEnd Pos
	Dictionary   *TableIdentifier
	Type         string
}

func (s *SystemReloadExpr) Pos() Pos {
	return s.ReloadPos
}

func (s *SystemReloadExpr) End() Pos {
	return s.StatementEnd
}

func (s *SystemReloadExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("RELOAD ")
	builder.WriteString(s.Type)
	if s.Dictionary != nil {
		builder.WriteByte(' ')
		builder.WriteString(s.Dictionary.String(level))
	}
	return builder.String()
}

func (s *SystemReloadExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if s.Dictionary != nil {
		if err := s.Dictionary.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSystemReloadExpr(s)
}

type SystemSyncExpr struct {
	SyncPos Pos
	Cluster *TableIdentifier
}

func (s *SystemSyncExpr) Pos() Pos {
	return s.SyncPos
}

func (s *SystemSyncExpr) End() Pos {
	return s.Cluster.End()
}

func (s *SystemSyncExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("SYNC ")
	builder.WriteString(s.Cluster.String(level))
	return builder.String()
}

func (s *SystemSyncExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Cluster.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitSystemSyncExpr(s)
}

type SystemCtrlExpr struct {
	CtrlPos      Pos
	StatementEnd Pos
	Command      string // START, STOP
	Type         string // REPLICATED, DISTRIBUTED
	Cluster      *TableIdentifier
}

func (s *SystemCtrlExpr) Pos() Pos {
	return s.CtrlPos
}

func (s *SystemCtrlExpr) End() Pos {
	return s.StatementEnd
}

func (s *SystemCtrlExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString(s.Command)
	builder.WriteByte(' ')
	builder.WriteString(s.Type)
	if s.Cluster != nil {
		builder.WriteByte(' ')
		builder.WriteString(s.Cluster.String(level))
	}
	return builder.String()
}

func (s *SystemCtrlExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if s.Cluster != nil {
		if err := s.Cluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSystemCtrlExpr(s)
}

type SystemDropExpr struct {
	DropPos      Pos
	StatementEnd Pos
	Type         string
}

func (s *SystemDropExpr) Pos() Pos {
	return s.DropPos
}

func (s *SystemDropExpr) End() Pos {
	return s.StatementEnd
}

func (s *SystemDropExpr) String(level int) string {
	return "DROP " + s.Type
}

func (s *SystemDropExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	return visitor.VisitSystemDropExpr(s)
}

type TruncateTable struct {
	TruncatePos  Pos
	StatementEnd Pos
	IsTemporary  bool
	IfExists     bool
	Name         *TableIdentifier
	OnCluster    *OnClusterExpr
}

func (t *TruncateTable) Pos() Pos {
	return t.TruncatePos
}

func (t *TruncateTable) End() Pos {
	return t.StatementEnd
}

func (t *TruncateTable) Type() string {
	return "TRUNCATE TABLE"
}

func (t *TruncateTable) String(level int) string {
	var builder strings.Builder
	builder.WriteString("TRUNCATE ")
	if t.IsTemporary {
		builder.WriteString("TEMPORARY ")
	}
	builder.WriteString("TABLE ")
	if t.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(t.Name.String(level))
	if t.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(t.OnCluster.String(level))
	}
	return builder.String()
}

func (t *TruncateTable) Accept(visitor ASTVisitor) error {
	visitor.enter(t)
	defer visitor.leave(t)
	if err := t.Name.Accept(visitor); err != nil {
		return err
	}
	if t.OnCluster != nil {
		if err := t.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitTruncateTable(t)
}

type SampleRatioExpr struct {
	SamplePos Pos
	Ratio     *RatioExpr
	Offset    *RatioExpr
}

func (s *SampleRatioExpr) Pos() Pos {
	return s.SamplePos
}

func (s *SampleRatioExpr) End() Pos {
	if s.Offset != nil {
		return s.Offset.End()
	}
	return s.Ratio.End()
}

func (s *SampleRatioExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("SAMPLE ")
	builder.WriteString(s.Ratio.String(level))
	if s.Offset != nil {
		builder.WriteString(" OFFSET ")
		builder.WriteString(s.Offset.String(level))
	}
	return builder.String()
}

func (s *SampleRatioExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(s)
	defer visitor.leave(s)
	if err := s.Ratio.Accept(visitor); err != nil {
		return err
	}
	if s.Offset != nil {
		if err := s.Offset.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitSampleRatioExpr(s)
}

type DeleteFromExpr struct {
	DeletePos Pos
	Table     *TableIdentifier
	OnCluster *OnClusterExpr
	WhereExpr Expr
}

func (d *DeleteFromExpr) Pos() Pos {
	return d.DeletePos
}

func (d *DeleteFromExpr) End() Pos {
	return d.WhereExpr.End()
}

func (d *DeleteFromExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("DELETE FROM ")
	builder.WriteString(d.Table.String(level))
	if d.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(d.OnCluster.String(level))
	}
	if d.WhereExpr != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString("WHERE ")
		builder.WriteString(d.WhereExpr.String(level))
	}
	return builder.String()
}

func (d *DeleteFromExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(d)
	defer visitor.leave(d)
	if err := d.Table.Accept(visitor); err != nil {
		return err
	}
	if d.OnCluster != nil {
		if err := d.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	if d.WhereExpr != nil {
		if err := d.WhereExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitDeleteFromExpr(d)
}

type ColumnNamesExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	ColumnNames   []NestedIdentifier
}

func (c *ColumnNamesExpr) Pos() Pos {
	return c.LeftParenPos
}

func (c *ColumnNamesExpr) End() Pos {
	return c.RightParenPos
}

func (c *ColumnNamesExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteByte('(')
	for i, column := range c.ColumnNames {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(column.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (c *ColumnNamesExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	for i := range c.ColumnNames {
		if err := c.ColumnNames[i].Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitColumnNamesExpr(c)
}

type ValuesExpr struct {
	LeftParenPos  Pos
	RightParenPos Pos
	Values        []Expr
}

func (v *ValuesExpr) Pos() Pos {
	return v.LeftParenPos
}

func (v *ValuesExpr) End() Pos {
	return v.RightParenPos
}

func (v *ValuesExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteByte('(')
	for i, value := range v.Values {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(value.String(level))
	}
	builder.WriteByte(')')
	return builder.String()
}

func (v *ValuesExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(v)
	defer visitor.leave(v)
	for _, value := range v.Values {
		if err := value.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitValuesExpr(v)
}

type InsertExpr struct {
	InsertPos   Pos
	Format      *FormatExpr
	Table       Expr
	ColumnNames *ColumnNamesExpr
	Values      []*ValuesExpr
	SelectExpr  *SelectQuery
}

func (i *InsertExpr) Pos() Pos {
	return i.InsertPos
}

func (i *InsertExpr) End() Pos {
	if i.SelectExpr != nil {
		return i.SelectExpr.End()
	}
	return i.Values[len(i.Values)-1].End()
}

func (i *InsertExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("INSERT INTO TABLE ")
	builder.WriteString(i.Table.String(level))
	if i.ColumnNames != nil {
		builder.WriteString(NewLine(level + 1))
		builder.WriteString(i.ColumnNames.String(level))
	}
	if i.Format != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(i.Format.String(level))
	}

	if i.SelectExpr != nil {
		builder.WriteString(i.SelectExpr.String(level))
	} else {
		builder.WriteString(NewLine(level))
		builder.WriteString("VALUES ")
		for j, value := range i.Values {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(NewLine(level + 1))
			builder.WriteString(value.String(level))
		}
	}
	return builder.String()
}

func (i *InsertExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(i)
	defer visitor.leave(i)
	if i.Format != nil {
		if err := i.Format.Accept(visitor); err != nil {
			return err
		}
	}
	if err := i.Table.Accept(visitor); err != nil {
		return err
	}
	if i.ColumnNames != nil {
		if err := i.ColumnNames.Accept(visitor); err != nil {
			return err
		}
	}
	for _, value := range i.Values {
		if err := value.Accept(visitor); err != nil {
			return err
		}
	}
	if i.SelectExpr != nil {
		if err := i.SelectExpr.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitInsertExpr(i)
}

type CheckExpr struct {
	CheckPos  Pos
	Table     *TableIdentifier
	Partition *PartitionExpr
}

func (c *CheckExpr) Pos() Pos {
	return c.CheckPos
}

func (c *CheckExpr) End() Pos {
	return c.Partition.End()
}

func (c *CheckExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("CHECK TABLE ")
	builder.WriteString(c.Table.String(level))
	builder.WriteString(NewLine(level))
	if c.Partition != nil {
		builder.WriteString(c.Partition.String(level))
	}
	return builder.String()
}

func (c *CheckExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(c)
	defer visitor.leave(c)
	if err := c.Table.Accept(visitor); err != nil {
		return err
	}
	if c.Partition != nil {
		if err := c.Partition.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitCheckExpr(c)
}

type UnaryExpr struct {
	UnaryPos Pos
	Kind     TokenKind
	Expr     Expr
}

func (n *UnaryExpr) Pos() Pos {
	return n.UnaryPos
}

func (n *UnaryExpr) End() Pos {
	return n.Expr.End()
}

func (n *UnaryExpr) String(level int) string {
	return "-" + n.Expr.String(level+1)
}

func (n *UnaryExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(n)
	defer visitor.leave(n)
	if err := n.Expr.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitUnaryExpr(n)
}

type RenameStmt struct {
	RenamePos    Pos
	StatementEnd Pos

	RenameTarget   string
	TargetPairList []*TargetPair
	OnCluster      *OnClusterExpr
}

func (r *RenameStmt) Pos() Pos {
	return r.RenamePos
}

func (r *RenameStmt) End() Pos {
	return r.StatementEnd
}

func (r *RenameStmt) Type() string {
	return "RENAME " + r.RenameTarget
}

func (r *RenameStmt) String(level int) string {
	var builder strings.Builder
	builder.WriteString("RENAME " + r.RenameTarget + " ")
	for i, pair := range r.TargetPairList {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(pair.Old.String(level))
		builder.WriteString(" TO ")
		builder.WriteString(pair.New.String(level))
	}
	if r.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(r.OnCluster.String(level))
	}
	return builder.String()
}

func (r *RenameStmt) Accept(visitor ASTVisitor) error {
	visitor.enter(r)
	defer visitor.leave(r)
	for _, pair := range r.TargetPairList {
		if err := pair.Old.Accept(visitor); err != nil {
			return err
		}
		if err := pair.New.Accept(visitor); err != nil {
			return err
		}
	}
	if r.OnCluster != nil {
		if err := r.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitRenameStmt(r)
}

type TargetPair struct {
	Old *TableIdentifier
	New *TableIdentifier
}

func (t *TargetPair) Pos() Pos {
	return t.Old.Pos()
}

func (t *TargetPair) End() Pos {
	return t.New.End()
}

func (t *TargetPair) String() string {
	return t.Old.String(0) + " TO " + t.New.String(0)
}

type ExplainExpr struct {
	ExplainPos Pos
	Type       string
	Statement  Expr
}

func (e *ExplainExpr) Pos() Pos {
	return e.ExplainPos
}

func (e *ExplainExpr) End() Pos {
	return e.Statement.End()
}

func (e *ExplainExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("EXPLAIN ")
	builder.WriteString(e.Type)
	builder.WriteByte(' ')
	builder.WriteString(e.Statement.String(level))
	return builder.String()
}

func (e *ExplainExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(e)
	defer visitor.leave(e)
	if err := e.Statement.Accept(visitor); err != nil {
		return err
	}
	return visitor.VisitExplainExpr(e)
}

type PrivilegeExpr struct {
	PrivilegePos Pos
	PrivilegeEnd Pos
	Keywords     []string
	Params       *ParamExprList
}

func (p *PrivilegeExpr) Pos() Pos {
	return p.PrivilegePos
}

func (p *PrivilegeExpr) End() Pos {
	return p.PrivilegeEnd
}

func (p *PrivilegeExpr) String(level int) string {
	var builder strings.Builder
	for i, keyword := range p.Keywords {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(keyword)
	}
	if p.Params != nil {
		builder.WriteString(p.Params.String(level))
	}
	return builder.String()
}

func (p *PrivilegeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(p)
	defer visitor.leave(p)
	if p.Params != nil {
		if err := p.Params.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitPrivilegeExpr(p)
}

type GrantPrivilegeExpr struct {
	GrantPos     Pos
	StatementEnd Pos
	OnCluster    *OnClusterExpr
	Privileges   []*PrivilegeExpr
	On           *TableIdentifier
	To           []*Ident
	WithOptions  []string
}

func (g *GrantPrivilegeExpr) Pos() Pos {
	return g.GrantPos
}

func (g *GrantPrivilegeExpr) End() Pos {
	return g.StatementEnd
}

func (g *GrantPrivilegeExpr) Type() string {
	return "GRANT PRIVILEGE"
}

func (g *GrantPrivilegeExpr) String(level int) string {
	var builder strings.Builder
	builder.WriteString("GRANT ")
	if g.OnCluster != nil {
		builder.WriteString(NewLine(level))
		builder.WriteString(g.OnCluster.String(level))
	}
	for i, privilege := range g.Privileges {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(privilege.String(level))
	}
	builder.WriteString(" ON ")
	builder.WriteString(g.On.String(level))
	builder.WriteString(" TO ")
	for i, role := range g.To {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(role.String(level))
	}
	for _, option := range g.WithOptions {
		builder.WriteString(" WITH " + option + " OPTION")
	}

	return builder.String()
}

func (g *GrantPrivilegeExpr) Accept(visitor ASTVisitor) error {
	visitor.enter(g)
	defer visitor.leave(g)
	if g.OnCluster != nil {
		if err := g.OnCluster.Accept(visitor); err != nil {
			return err
		}
	}
	for _, privilege := range g.Privileges {
		if err := privilege.Accept(visitor); err != nil {
			return err
		}
	}
	if err := g.On.Accept(visitor); err != nil {
		return err
	}
	for _, role := range g.To {
		if err := role.Accept(visitor); err != nil {
			return err
		}
	}
	return visitor.VisitGrantPrivilegeExpr(g)
}
