// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package expression

import (
	"github.com/juju/errors"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/types/json"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tipb/go-tipb"
)

var (
	_ functionClass = &jsonTypeFunctionClass{}
	_ functionClass = &jsonExtractFunctionClass{}
	_ functionClass = &jsonUnquoteFunctionClass{}
	_ functionClass = &jsonSetFunctionClass{}
	_ functionClass = &jsonInsertFunctionClass{}
	_ functionClass = &jsonReplaceFunctionClass{}
	_ functionClass = &jsonRemoveFunctionClass{}
	_ functionClass = &jsonMergeFunctionClass{}
	_ functionClass = &jsonObjectFunctionClass{}
	_ functionClass = &jsonArrayFunctionClass{}
	_ functionClass = &jsonContainsFunctionClass{}

	// Type of JSON value.
	_ builtinFunc = &builtinJSONTypeSig{}
	// Unquote JSON value.
	_ builtinFunc = &builtinJSONUnquoteSig{}
	// Create JSON array.
	_ builtinFunc = &builtinJSONArraySig{}
	// Create JSON object.
	_ builtinFunc = &builtinJSONObjectSig{}
	// Return data from JSON document.
	_ builtinFunc = &builtinJSONExtractSig{}
	// Insert data into JSON document.
	_ builtinFunc = &builtinJSONSetSig{}
	// Insert data into JSON document.
	_ builtinFunc = &builtinJSONInsertSig{}
	// Replace values in JSON document.
	_ builtinFunc = &builtinJSONReplaceSig{}
	// Remove data from JSON document.
	_ builtinFunc = &builtinJSONRemoveSig{}
	// Merge JSON documents, preserving duplicate keys.
	_ builtinFunc = &builtinJSONMergeSig{}
	// Check JSON document contains specific target.
	_ builtinFunc = &builtinJSONContainsSig{}
)

type jsonTypeFunctionClass struct {
	baseFunctionClass
}

type builtinJSONTypeSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONTypeSig) Clone() builtinFunc {
	newSig := &builtinJSONTypeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonTypeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETString, types.ETJson)
	bf.tp.Flen = 51 // Flen of JSON_TYPE is length of UNSIGNED INTEGER.
	sig := &builtinJSONTypeSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonTypeSig)
	return sig, nil
}

func (b *builtinJSONTypeSig) evalString(row chunk.Row) (res string, isNull bool, err error) {
	var j json.BinaryJSON
	j, isNull, err = b.args[0].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, errors.Trace(err)
	}
	return j.Type(), false, nil
}

type jsonExtractFunctionClass struct {
	baseFunctionClass
}

type builtinJSONExtractSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONExtractSig) Clone() builtinFunc {
	newSig := &builtinJSONExtractSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonExtractFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]types.EvalType, 0, len(args))
	argTps = append(argTps, types.ETJson)
	for range args[1:] {
		argTps = append(argTps, types.ETString)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	sig := &builtinJSONExtractSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonExtractSig)
	return sig, nil
}

func (b *builtinJSONExtractSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = b.args[0].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return
	}
	pathExprs := make([]json.PathExpression, 0, len(b.args)-1)
	for _, arg := range b.args[1:] {
		var s string
		s, isNull, err = arg.EvalString(b.ctx, row)
		if isNull || err != nil {
			return res, isNull, errors.Trace(err)
		}
		pathExpr, err := json.ParseJSONPathExpr(s)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		pathExprs = append(pathExprs, pathExpr)
	}
	var found bool
	if res, found = res.Extract(pathExprs); !found {
		return res, true, nil
	}
	return res, false, nil
}

type jsonUnquoteFunctionClass struct {
	baseFunctionClass
}

type builtinJSONUnquoteSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONUnquoteSig) Clone() builtinFunc {
	newSig := &builtinJSONUnquoteSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonUnquoteFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETString, types.ETJson)
	args[0].GetType().Flag &= ^mysql.ParseToJSONFlag
	sig := &builtinJSONUnquoteSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonUnquoteSig)
	return sig, nil
}

func (b *builtinJSONUnquoteSig) evalString(row chunk.Row) (res string, isNull bool, err error) {
	var j json.BinaryJSON
	j, isNull, err = b.args[0].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, errors.Trace(err)
	}
	res, err = j.Unquote()
	return res, err != nil, errors.Trace(err)
}

type jsonSetFunctionClass struct {
	baseFunctionClass
}

type builtinJSONSetSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONSetSig) Clone() builtinFunc {
	newSig := &builtinJSONSetSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonSetFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	if len(args)&1 != 1 {
		return nil, ErrIncorrectParameterCount.GenByArgs(c.funcName)
	}
	argTps := make([]types.EvalType, 0, len(args))
	argTps = append(argTps, types.ETJson)
	for i := 1; i < len(args)-1; i += 2 {
		argTps = append(argTps, types.ETString, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	for i := 2; i < len(args); i += 2 {
		args[i].GetType().Flag &= ^mysql.ParseToJSONFlag
	}
	sig := &builtinJSONSetSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonSetSig)
	return sig, nil
}

func (b *builtinJSONSetSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = jsonModify(b.ctx, b.args, row, json.ModifySet)
	return res, isNull, errors.Trace(err)
}

type jsonInsertFunctionClass struct {
	baseFunctionClass
}

type builtinJSONInsertSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONInsertSig) Clone() builtinFunc {
	newSig := &builtinJSONInsertSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonInsertFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	if len(args)&1 != 1 {
		return nil, ErrIncorrectParameterCount.GenByArgs(c.funcName)
	}
	argTps := make([]types.EvalType, 0, len(args))
	argTps = append(argTps, types.ETJson)
	for i := 1; i < len(args)-1; i += 2 {
		argTps = append(argTps, types.ETString, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	for i := 2; i < len(args); i += 2 {
		args[i].GetType().Flag &= ^mysql.ParseToJSONFlag
	}
	sig := &builtinJSONInsertSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonInsertSig)
	return sig, nil
}

func (b *builtinJSONInsertSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = jsonModify(b.ctx, b.args, row, json.ModifyInsert)
	return res, isNull, errors.Trace(err)
}

type jsonReplaceFunctionClass struct {
	baseFunctionClass
}

type builtinJSONReplaceSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONReplaceSig) Clone() builtinFunc {
	newSig := &builtinJSONReplaceSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonReplaceFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	if len(args)&1 != 1 {
		return nil, ErrIncorrectParameterCount.GenByArgs(c.funcName)
	}
	argTps := make([]types.EvalType, 0, len(args))
	argTps = append(argTps, types.ETJson)
	for i := 1; i < len(args)-1; i += 2 {
		argTps = append(argTps, types.ETString, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	for i := 2; i < len(args); i += 2 {
		args[i].GetType().Flag &= ^mysql.ParseToJSONFlag
	}
	sig := &builtinJSONReplaceSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonReplaceSig)
	return sig, nil
}

func (b *builtinJSONReplaceSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = jsonModify(b.ctx, b.args, row, json.ModifyReplace)
	return res, isNull, errors.Trace(err)
}

type jsonRemoveFunctionClass struct {
	baseFunctionClass
}

type builtinJSONRemoveSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONRemoveSig) Clone() builtinFunc {
	newSig := &builtinJSONRemoveSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonRemoveFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]types.EvalType, 0, len(args))
	argTps = append(argTps, types.ETJson)
	for range args[1:] {
		argTps = append(argTps, types.ETString)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	sig := &builtinJSONRemoveSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonRemoveSig)
	return sig, nil
}

func (b *builtinJSONRemoveSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = b.args[0].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return res, isNull, errors.Trace(err)
	}
	pathExprs := make([]json.PathExpression, 0, len(b.args)-1)
	for _, arg := range b.args[1:] {
		var s string
		s, isNull, err = arg.EvalString(b.ctx, row)
		if isNull || err != nil {
			return res, isNull, errors.Trace(err)
		}
		var pathExpr json.PathExpression
		pathExpr, err = json.ParseJSONPathExpr(s)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		pathExprs = append(pathExprs, pathExpr)
	}
	res, err = res.Remove(pathExprs)
	if err != nil {
		return res, true, errors.Trace(err)
	}
	return res, false, nil
}

type jsonMergeFunctionClass struct {
	baseFunctionClass
}

type builtinJSONMergeSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONMergeSig) Clone() builtinFunc {
	newSig := &builtinJSONMergeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonMergeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]types.EvalType, 0, len(args))
	for range args {
		argTps = append(argTps, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	sig := &builtinJSONMergeSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonMergeSig)
	return sig, nil
}

func (b *builtinJSONMergeSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	values := make([]json.BinaryJSON, 0, len(b.args))
	for _, arg := range b.args {
		var value json.BinaryJSON
		value, isNull, err = arg.EvalJSON(b.ctx, row)
		if isNull || err != nil {
			return res, isNull, errors.Trace(err)
		}
		values = append(values, value)
	}
	res = json.MergeBinary(values)
	return res, false, nil
}

type jsonObjectFunctionClass struct {
	baseFunctionClass
}

type builtinJSONObjectSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONObjectSig) Clone() builtinFunc {
	newSig := &builtinJSONObjectSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonObjectFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	if len(args)&1 != 0 {
		return nil, ErrIncorrectParameterCount.GenByArgs(c.funcName)
	}
	argTps := make([]types.EvalType, 0, len(args))
	for i := 0; i < len(args)-1; i += 2 {
		argTps = append(argTps, types.ETString, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	for i := 1; i < len(args); i += 2 {
		args[i].GetType().Flag &= ^mysql.ParseToJSONFlag
	}
	sig := &builtinJSONObjectSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonObjectSig)
	return sig, nil
}

func (b *builtinJSONObjectSig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	if len(b.args)&1 == 1 {
		err = ErrIncorrectParameterCount.GenByArgs(ast.JSONObject)
		return res, true, errors.Trace(err)
	}
	jsons := make(map[string]interface{}, len(b.args)>>1)
	var key string
	var value json.BinaryJSON
	for i, arg := range b.args {
		if i&1 == 0 {
			key, isNull, err = arg.EvalString(b.ctx, row)
			if err != nil {
				return res, true, errors.Trace(err)
			}
			if isNull {
				err = errors.New("JSON documents may not contain NULL member names")
				return res, true, errors.Trace(err)
			}
		} else {
			value, isNull, err = arg.EvalJSON(b.ctx, row)
			if err != nil {
				return res, true, errors.Trace(err)
			}
			if isNull {
				value = json.CreateBinary(nil)
			}
			jsons[key] = value
		}
	}
	return json.CreateBinary(jsons), false, nil
}

type jsonArrayFunctionClass struct {
	baseFunctionClass
}

type builtinJSONArraySig struct {
	baseBuiltinFunc
}

func (b *builtinJSONArraySig) Clone() builtinFunc {
	newSig := &builtinJSONArraySig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonArrayFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]types.EvalType, 0, len(args))
	for range args {
		argTps = append(argTps, types.ETJson)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETJson, argTps...)
	for i := range args {
		args[i].GetType().Flag &= ^mysql.ParseToJSONFlag
	}
	sig := &builtinJSONArraySig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonArraySig)
	return sig, nil
}

func (b *builtinJSONArraySig) evalJSON(row chunk.Row) (res json.BinaryJSON, isNull bool, err error) {
	jsons := make([]interface{}, 0, len(b.args))
	for _, arg := range b.args {
		j, isNull, err := arg.EvalJSON(b.ctx, row)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		if isNull {
			j = json.CreateBinary(nil)
		}
		jsons = append(jsons, j)
	}
	return json.CreateBinary(jsons), false, nil
}

func jsonModify(ctx sessionctx.Context, args []Expression, row chunk.Row, mt json.ModifyType) (res json.BinaryJSON, isNull bool, err error) {
	res, isNull, err = args[0].EvalJSON(ctx, row)
	if isNull || err != nil {
		return res, isNull, errors.Trace(err)
	}
	pathExprs := make([]json.PathExpression, 0, (len(args)-1)/2+1)
	for i := 1; i < len(args); i += 2 {
		// TODO: We can cache pathExprs if args are constants.
		var s string
		s, isNull, err = args[i].EvalString(ctx, row)
		if isNull || err != nil {
			return res, isNull, errors.Trace(err)
		}
		var pathExpr json.PathExpression
		pathExpr, err = json.ParseJSONPathExpr(s)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		pathExprs = append(pathExprs, pathExpr)
	}
	values := make([]json.BinaryJSON, 0, (len(args)-1)/2+1)
	for i := 2; i < len(args); i += 2 {
		var value json.BinaryJSON
		value, isNull, err = args[i].EvalJSON(ctx, row)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		if isNull {
			value = json.CreateBinary(nil)
		}
		values = append(values, value)
	}
	res, err = res.Modify(pathExprs, values, mt)
	if err != nil {
		return res, true, errors.Trace(err)
	}
	return res, false, nil
}

type jsonContainsFunctionClass struct {
	baseFunctionClass
}

type builtinJSONContainsSig struct {
	baseBuiltinFunc
}

func (b *builtinJSONContainsSig) Clone() builtinFunc {
	newSig := &builtinJSONContainsSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (c *jsonContainsFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := []types.EvalType{types.ETJson, types.ETJson}
	if len(args) == 3 {
		argTps = append(argTps, types.ETString)
	}
	bf := newBaseBuiltinFuncWithTp(ctx, args, types.ETInt, argTps...)
	sig := &builtinJSONContainsSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_JsonContainsSig)
	return sig, nil
}

func (b *builtinJSONContainsSig) evalInt(row chunk.Row) (res int64, isNull bool, err error) {
	obj, isNull, err := b.args[0].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return res, isNull, errors.Trace(err)
	}
	target, isNull, err := b.args[1].EvalJSON(b.ctx, row)
	if isNull || err != nil {
		return res, isNull, errors.Trace(err)
	}
	var pathExpr json.PathExpression
	if len(b.args) == 3 {
		path, isNull, err := b.args[2].EvalString(b.ctx, row)
		if isNull || err != nil {
			return res, isNull, errors.Trace(err)
		}
		pathExpr, err = json.ParseJSONPathExpr(path)
		if err != nil {
			return res, true, errors.Trace(err)
		}
		if pathExpr.ContainsAnyAsterisk() {
			return res, true, json.ErrInvalidJSONPathWildcard
		}
		var exists bool
		obj, exists = obj.Extract([]json.PathExpression{pathExpr})
		if !exists {
			return res, true, nil
		}
	}

	if json.ContainsBinary(obj, target) {
		return 1, false, nil
	}
	return 0, false, nil
}
