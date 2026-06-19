package dto

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
)

const (
	FilterOperatorEq        = "eq"
	FilterOperatorLike      = "like"
	FilterOperatorIn        = "in"
	FilterOperatorNotEq     = "not_eq"
	FilterOperatorLessEq    = "less_eq"
	FilterOperatorGreaterEq = "greater_eq"
	FilterPlainQuery        = "plan"
	FilterIsNotNull         = "is_not_null"
	FilterIsNull            = "is_null"
)

const (
	FilterGroupOperatorAnd = "AND"
	FilterGroupOperatorOr  = "OR"
)

type Filter struct {
	ArgName  string
	Field    string
	Value    any
	Operator string `validate:"required,oneof=eq like in not_eq less_eq greater_eq"`
	Table    string
}

func (f *Filter) GetWhereClause() (string, map[string]any) {
	args := map[string]any{}

	column := f.Field
	if f.Table != "" {
		column = fmt.Sprintf("%s.%s", f.Table, f.Field)
	}

	argName := f.ArgName
	if argName == "" {
		argName = f.Field
	}

	switch f.Operator {
	case FilterOperatorEq:
		args[argName] = f.Value

		return fmt.Sprintf("%s = :%s", column, argName), args
	case FilterOperatorLike:
		args[argName] = fmt.Sprintf("%%%s%%", f.Value)

		return fmt.Sprintf("LOWER(%s) LIKE LOWER(:%s) ", column, argName), args
	case FilterOperatorIn:
		val := reflect.ValueOf(f.Value)
		vType := val.Type()

		switch vType.Kind() {
		case reflect.Array, reflect.Slice:
			named := make([]string, val.Len())

			for idx := range val.Len() {
				args[fmt.Sprintf("%s_%d", argName, idx)] = val.Index(idx).Interface()

				named[idx] = fmt.Sprintf(":%s_%d", argName, idx)
			}

			return fmt.Sprintf("%s IN (%s) ", column, strings.Join(named, ", ")), args
		default:
			return fmt.Sprintf("%s IN (%s) ", column, f.Value), args
		}
	case FilterOperatorNotEq:
		args[argName] = f.Value

		return fmt.Sprintf("%s != :%s", column, argName), args
	case FilterOperatorLessEq:
		args[argName] = f.Value

		return fmt.Sprintf("%s <= :%s", column, argName), args
	case FilterOperatorGreaterEq:
		args[argName] = f.Value

		return fmt.Sprintf("%s >= :%s", column, argName), args
	case FilterPlainQuery:
		query, _ := f.Value.(string)

		return fmt.Sprintf("(%s)", query), args
	case FilterIsNotNull:
		return column + " IS NOT NULL", args
	case FilterIsNull:
		return column + " IS NULL", args
	default:
		return "", args
	}
}

type FilterGroup struct {
	Filters  []any
	Operator string
}

func (f *FilterGroup) GetWhereClause() (string, map[string]any) {
	args := map[string]any{}
	whereClause := []string{}

	for _, filter := range f.Filters {
		switch fill := filter.(type) {
		case Filter:
			where, arg := fill.GetWhereClause()
			whereClause = append(whereClause, where)

			maps.Copy(args, arg)
		case FilterGroup:
			where, arg := fill.GetWhereClause()
			whereClause = append(whereClause, where)

			maps.Copy(args, arg)
		}
	}

	if len(whereClause) == 0 {
		return "", args
	}

	return fmt.Sprintf("(%s)", strings.Join(whereClause, " "+f.Operator+" ")), args
}
