package schemamd

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

// childIsRequired returns true for blocks with min items > 0 or explicitly required
// attributes
func childIsRequired(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		return att.Required
	}

	return block.MinItems > 0
}

// childIsOptional returns true for blocks with with min items 0, but any required or
// optional children, or explicitly optional attributes
func childIsOptional(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		return att.Optional
	}

	if block.MinItems > 0 {
		return false
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if childIsRequired(childBlock, nil) {
			return true
		}
		if childIsOptional(childBlock, nil) {
			return true
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if childIsRequired(nil, childAtt) {
			return true
		}
		if childIsOptional(nil, childAtt) {
			return true
		}
	}

	return false
}

// childIsReadOnly returns true for blocks where all leaves are read only (computed
// but not optional)
func childIsReadOnly(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		// these shouldn't be able to be required, but just in case
		return att.Computed && !att.Optional && !att.Required
	}

	if block.MinItems != 0 || block.MaxItems != 0 {
		return false
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if !childIsReadOnly(childBlock, nil) {
			return false
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if !childIsReadOnly(nil, childAtt) {
			return false
		}
	}

	return true
}

func typeString(ty cty.Type) string {
	// Easy cases first
	switch ty {
	case cty.String:
		return "string"
	case cty.Bool:
		return "bool"
	case cty.Number:
		return "number"
	case cty.DynamicPseudoType:
		return "any"
	}

	if ty.IsCapsuleType() {
		panic("typeString does not support capsule types")
	}

	if ty.IsCollectionType() {
		ety := ty.ElementType()
		etyString := typeString(ety)
		switch {
		case ty.IsListType():
			return fmt.Sprintf("list(%s)", etyString)
		case ty.IsSetType():
			return fmt.Sprintf("set(%s)", etyString)
		case ty.IsMapType():
			return fmt.Sprintf("map(%s)", etyString)
		default:
			// Should never happen because the above is exhaustive
			panic("unsupported collection type")
		}
	}

	if ty.IsObjectType() {
		var buf bytes.Buffer
		buf.WriteString("object({")
		atys := ty.AttributeTypes()
		names := make([]string, 0, len(atys))
		for name := range atys {
			names = append(names, name)
		}
		sort.Strings(names)
		first := true
		for _, name := range names {
			aty := atys[name]
			if !first {
				buf.WriteByte(',')
				buf.WriteByte('\n')
			}
			if !hclsyntax.ValidIdentifier(name) {
				// Should never happen for any type produced by this package,
				// but we'll do something reasonable here just so we don't
				// produce garbage if someone gives us a hand-assembled object
				// type that has weird attribute names.
				// Using Go-style quoting here isn't perfect, since it doesn't
				// exactly match HCL syntax, but it's fine for an edge-case.
				buf.WriteString(fmt.Sprintf("%q", name))
			} else {
				buf.WriteString(name)
			}
			buf.WriteByte('=')
			buf.WriteString(typeString(aty))
			first = false
		}
		buf.WriteString("})")
		return buf.String()
	}

	if ty.IsTupleType() {
		var buf bytes.Buffer
		buf.WriteString("tuple([")
		etys := ty.TupleElementTypes()
		first := true
		for _, ety := range etys {
			if !first {
				buf.WriteByte(',')
			}
			buf.WriteString(typeString(ety))
			first = false
		}
		buf.WriteString("])")
		return buf.String()
	}

	// Should never happen because we covered all cases above.
	panic(fmt.Errorf("unsupported type %#v", ty))
}
