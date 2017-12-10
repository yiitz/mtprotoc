/*
 *  Copyright (c) 2017, https://github.com/nebulaim
 *  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gengolang

import (
	"fmt"
	mtproto_parser "github.com/nebulaim/mtprotoc/codegen/parser"
	"strconv"
	"strings"
)

// types
type TplTypesDataList struct {
	BaseTypeList []TplBaseTypeData
}

// functions
type TplFunctionDataList struct {
	RequestList []TplMessageData
	// Vector是非标准proto消息，故要自动生成一个Vector包装proto消息
	// 注意去重
	VectorResList []TplParam
	// RpcList
	// service RPCAuth {
	//	rpc auth_checkPhone(TL_auth_checkPhone) returns (auth_CheckedPhone) {}
	// }
	ServiceList []TplBaseTypeData
}

// 参数列表
type TplParam struct {
	Type  string
	Name  string
	Name2 string
	Index int
}

// 对应生成proto消息
type TplMessageData struct {
	Predicate string
	Name      string
	// 碰撞的字段名，特殊处理
	ParamList []TplParam
	ResType   string
	Line      string

	EncodeCodeList []string
	DecodeCodeList []string
}

// Base类型
type TplBaseTypeData struct {
	Name      string
	ParamList []TplParam
	// Line string
	ResType string
	// 所有的子类
	SubMessageList []TplMessageData
}

func toMessageName(n string) string {
	return strings.Replace(n, ".", "_", -1)
}

var (
	ignoreRpcList = []string{"invokeAfterMsg", "invokeAfterMsgs", "initConnection", "invokeWithLayer", "invokeWithoutUpdates"}
)

func toGolangType(t mtproto_parser.Type) (n string) {
	switch t.(type) {
	case mtproto_parser.BoolType:
		n = "bool"
	case mtproto_parser.IntType:
		n = "int32"
	case mtproto_parser.LongType:
		n = "int64"
	case mtproto_parser.DoubleType:
		n = "float64"
	case mtproto_parser.Int128Type:
		n = "[]byte"
	case mtproto_parser.Int256Type:
		n = "[]byte"
	case mtproto_parser.StringType:
		n = "string"
	case mtproto_parser.BytesType:
		n = "[]byte"
	case mtproto_parser.FlagsType:
		n = ""
	case mtproto_parser.SubFlagsType:
		t2, _ := t.(mtproto_parser.SubFlagsType)
		// glog.Info(t2)
		n = toGolangType(t2.Type)
	case mtproto_parser.BuiltInVectorType:
		t2, _ := t.(mtproto_parser.BuiltInVectorType)
		n = fmt.Sprintf("[]%s", toGolangType(t2.Type))
	case mtproto_parser.TVectorType:
		t2, _ := t.(mtproto_parser.TVectorType)
		n = fmt.Sprintf("[]%s", toGolangType(t2.Type))
	case mtproto_parser.CustomType:
		n = "*" + toProtoGoName(strings.Replace(t.Name(), ".", "_", -1))
		//if n == "Bool" {
		//	n = "bool"
		//}
	case mtproto_parser.Constructor:
		t2, _ := t.(mtproto_parser.Constructor)
		n = "*TL" + toProtoGoName(strings.Replace(t2.Predicate, ".", "_", -1))
	case mtproto_parser.TemplateType:
		n = "[]byte"
	default:
		panic(fmt.Errorf("Bad type: %v", t))
	}

	return
}

func toProtoGoName(n string) string {
	if len(n) == 0 {
		return n
	}
	var n2 = n

	// TODO(@benqi): add ruler table
	if n == "udp_p2p" {
		n2 = "UdpP2P"
	}
	ss := strings.Split(n2, "_")
	for i, v := range ss {
		if i != 0 && IsUpper(v[0]) {
			ss[i] = "_" + v
		} else {
			// glog.Info(v)
			ss[i] = string(ToUpper(v[0])) + v[1:]
		}
	}
	return strings.Join(ss, "")
}

func findByParamList(params []TplParam, p2 TplParam) int {
	for i, p := range params {
		if p.Name == p2.Name && p.Type == p2.Type {
			return i
		}
	}
	return -1
}

func makeEncodeFlags(params []mtproto_parser.Param) (s string) {
	s = fmt.Sprintf("// flags\n")
	s += fmt.Sprintf("    var flags uint32 = 0\n")
	for i, p := range params {
		switch p.Type.(type) {
		case mtproto_parser.SubFlagsType:
			t2, _ := p.Type.(mtproto_parser.SubFlagsType)
			// TODO(@benqi): other type
			switch t2.Type.(type) {
			case mtproto_parser.BoolType:
				s += fmt.Sprintf("    if m.Get%s() == true { flags |= 1 << %d }\n", toProtoGoName(p.Name), i)
			case mtproto_parser.IntType, mtproto_parser.LongType:
				s += fmt.Sprintf("    if m.Get%s() != 0 { flags |= 1 << %d }\n", toProtoGoName(p.Name), i)
			case mtproto_parser.StringType:
				s += fmt.Sprintf("    if m.Get%s() != \"\" { flags |= 1 << %d }\n", toProtoGoName(p.Name), i)
			default:
				s += fmt.Sprintf("    if m.Get%s() != nil { flags |= 1 << %d }\n", toProtoGoName(p.Name), i)
			}
		}
	}
	s += fmt.Sprintf("	x.UInt(flags)\n")
	return
}

func makeCodecCodeByNameType(n string, t mtproto_parser.Type, idx int) (e string, d string) {
	switch t.(type) {
	case mtproto_parser.BoolType:
		// e = fmt.Sprintf("// x.Bool()")
		d = fmt.Sprintf("(true)", toProtoGoName(n))
	case mtproto_parser.IntType:
		e = fmt.Sprintf("x.Int(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.Int())", toProtoGoName(n))
	case mtproto_parser.LongType:
		e = fmt.Sprintf("x.Long(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.Long())", toProtoGoName(n))
	case mtproto_parser.DoubleType:
		e = fmt.Sprintf("x.Double(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.Double())", toProtoGoName(n))
	case mtproto_parser.Int128Type, mtproto_parser.Int256Type, mtproto_parser.BytesType:
		e = fmt.Sprintf("x.Bytes(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.Bytes())", toProtoGoName(n))
	case mtproto_parser.StringType:
		e = fmt.Sprintf("x.StringBytes(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.StringBytes())", toProtoGoName(n))
	case mtproto_parser.Constructor:
		e = fmt.Sprintf("x.Bytes(m.Get%s().Encode())", toProtoGoName(n))
		d = fmt.Sprintf("m%d := &%s{}\n    m%d.Decode(dbuf)\n    m.Set%s(m%d)", idx, toProtoGoName(n), idx, toProtoGoName(n), idx)
	case mtproto_parser.CustomType:
		e = fmt.Sprintf("x.Bytes(m.Get%s().Encode())", toProtoGoName(n))
		d = fmt.Sprintf("m%d := &%s{}\n    m%d.Decode(dbuf)\n    m.Set%s(m%d)", idx, toProtoGoName(n), idx, toProtoGoName(n), idx)
	}

	return
}

// Encode
// Decode
func makeCodecCode(params []mtproto_parser.Param, n string, t mtproto_parser.Type, idx int) (e string, d string) {
	switch t.(type) {
	case mtproto_parser.BoolType:
		// e = fmt.Sprintf("// x.Bool()")
		d = fmt.Sprintf("m.Data2.%s = true", toProtoGoName(n))
	case mtproto_parser.IntType:
		e = fmt.Sprintf("x.Int(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.Int()", toProtoGoName(n))
	case mtproto_parser.LongType:
		e = fmt.Sprintf("x.Long(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.Long()", toProtoGoName(n))
	case mtproto_parser.DoubleType:
		e = fmt.Sprintf("x.Double(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.Double()", toProtoGoName(n))
	case mtproto_parser.Int128Type:
		e = fmt.Sprintf("x.Bytes(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.Bytes()", toProtoGoName(n))
	case mtproto_parser.Int256Type:
		e = fmt.Sprintf("x.Bytes(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.Bytes()", toProtoGoName(n))
	case mtproto_parser.StringType:
		e = fmt.Sprintf("x.StringBytes(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.StringBytes()", toProtoGoName(n))
	case mtproto_parser.BytesType:
		e = fmt.Sprintf("x.StringBytes(m.Data2.%s)", toProtoGoName(n))
		d = fmt.Sprintf("m.Data2.%s = x.StringBytes()", toProtoGoName(n))
	case mtproto_parser.FlagsType:
		e = makeEncodeFlags(params)
		d = fmt.Sprintf("flags := dbuf.UInt()")
		// glog.Info(e, " ==> ", d)
	case mtproto_parser.SubFlagsType:
		t2, _ := t.(mtproto_parser.SubFlagsType)
		// TODO(@benqi): other type
		switch t2.Type.(type) {
		case mtproto_parser.BoolType:
			d = fmt.Sprintf("if (flags & (1 << %d)) != 0 { m.Data2.%s = true }", idx, toProtoGoName(n))
		case mtproto_parser.IntType,
			mtproto_parser.LongType,
			mtproto_parser.StringType,
			mtproto_parser.BytesType,
			mtproto_parser.Constructor,
			mtproto_parser.CustomType:
			e2, d2 := makeCodecCode(params, n, t2.Type, idx)
			e = fmt.Sprintf("if m.Get%s() != 0 { %s }", toProtoGoName(n), e2)
			d = fmt.Sprintf("if (flags & (1 << %d)) != 0 { %s }", idx, d2)
		case mtproto_parser.BuiltInVectorType, mtproto_parser.TVectorType:
			t2, _ := t.(mtproto_parser.BuiltInVectorType)
			e2, d2 := makeCodecCode(params, n, t2.Type, idx)
			e = fmt.Sprintf("if m.Data2.%s != 0 {\n %s \n}", toProtoGoName(n), e2)
			d = fmt.Sprintf("if (flags & (1 << %d)) != 0 {\n %s \n}", idx, d2)
		default:
		}

	case mtproto_parser.BuiltInVectorType:
		t2, _ := t.(mtproto_parser.BuiltInVectorType)
		n2 := toProtoGoName(n)
		switch t2.Type.(type) {
		case mtproto_parser.IntType:
			e = fmt.Sprintf("x.VectorInt(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorInt()", n2)
		case mtproto_parser.LongType:
			e = fmt.Sprintf("x.VectorLong(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorLong()", n2)
		case mtproto_parser.StringType:
			e = fmt.Sprintf("x.VectorString(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorString()", n2)
		case mtproto_parser.CustomType, mtproto_parser.Constructor:
			e = fmt.Sprintf("x.Int(int32(m.Data2.%s))\n", n2)
			e += fmt.Sprintf("for _, v: = range m.Data2.%s {\n", n2)
			e += fmt.Sprintf("  x.buf = append(x.buf, *v.Encode()...)")
			e += fmt.Sprintf("}", n2)
			d = fmt.Sprintf("ln := dbuf.Int()\n", n2)
			d = fmt.Sprintf("m.Data2.%s = make([]*%s, ln)", n2, toGolangType(t2.Type))
			d = fmt.Sprintf("for i < ln {\n m.Data2.%s[i] = &%s\n (*m.Data2.%s[i]).Decode(dbuf)\n}", n2, toGolangType(t2.Type)) // ln := len(m.Data2.%s)\n", n2)
		}
	case mtproto_parser.TVectorType:
		t2, _ := t.(mtproto_parser.BuiltInVectorType)
		n2 := toProtoGoName(n)
		switch t2.Type.(type) {
		case mtproto_parser.IntType:
			e = fmt.Sprintf("x.VectorInt(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorInt()", n2)
		case mtproto_parser.LongType:
			e = fmt.Sprintf("x.VectorLong(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorLong()", n2)
		case mtproto_parser.StringType:
			e = fmt.Sprintf("x.VectorString(m.Data2.%s)\n", n2)
			d = fmt.Sprintf("m.Data2.%s = x.VectorString()", n2)
		case mtproto_parser.CustomType, mtproto_parser.Constructor:
			e = "x.Int(int32(TLConstructor_CRC32_vector))"
			e += fmt.Sprintf("x.Int(int32(m.Data2.%s))\n", n2)
			e += fmt.Sprintf("for _, v: = range m.Data2.%s {\n", n2)
			e += fmt.Sprintf("  x.buf = append(x.buf, *v.Encode()...)", n2)
			e += fmt.Sprintf("}", n2)

			d = "dbuf.Int()\n"
			d += fmt.Sprintf("ln := dbuf.Int()\n", n2)
			d += fmt.Sprintf("m.Data2.%s = make([]*%s, ln)", n2, toGolangType(t2.Type))
			d += fmt.Sprintf("for i < ln {\n m.Data2.%s[i] = &%s\n (*m.Data2.%s[i]).Decode(dbuf)\n}", n2, toGolangType(t2.Type)) // ln := len(m.Data2.%s)\n", n2)
		}
	case mtproto_parser.CustomType, mtproto_parser.Constructor:
		e = fmt.Sprintf("x.Int(int32(TLConstructor_CRC32_%s)")
		e = fmt.Sprintf("x.Bytes(m.Data2%s.Encode())", toProtoGoName(n))
		d = fmt.Sprintf("m%d := &%s{}\n    m%d.Decode(dbuf)\n    m.Set%s(m%d)", idx, toProtoGoName(n), idx, toProtoGoName(n), idx)
	case mtproto_parser.TemplateType:
		// n = "[]byte"
		e = fmt.Sprintf("x.Bytes(m.Get%s())", toProtoGoName(n))
		d = fmt.Sprintf("m.Set%s(x.Bytes())", toProtoGoName(n))
	default:
	}
	return
}

func checkByStringList(strList []string, s string) bool {
	for _, p := range strList {
		if p == s {
			return true
		}
	}
	return false
}

func makeBaseTypeListTpl(schemas *mtproto_parser.Schemas) (types *TplTypesDataList) {
	baseTypeMap := make(map[string]*TplBaseTypeData)

	for _, c := range schemas.ConstructorList {
		baseTypeName := toProtoGoName(toMessageName(c.BaseType.Name()))
		// baseMessage := &TplBaseMessage{Name: baseTypeName}
		baseType, ok := baseTypeMap[baseTypeName]
		if !ok {
			baseType = &TplBaseTypeData{
				Name: baseTypeName,
			}
			baseTypeMap[baseTypeName] = baseType
		}

		messageData := TplMessageData{
			Predicate: toMessageName(c.Predicate),
			Name:      toProtoGoName(toMessageName(c.Predicate)),
			Line:      c.Line,
			ResType:   baseTypeName,
			// toMessageName(c.BaseType.Name()),
		}
		//glog.Info(c.Line)
		//
		for idx, p := range c.ParamList {
			param := TplParam{}
			param.Name = toProtoGoName(p.Name)
			param.Type = toGolangType(p.Type)

			// flags
			e, d := makeCodecCode(c.ParamList, p.Name, p.Type, idx+1)
			messageData.EncodeCodeList = append(messageData.EncodeCodeList, e)
			messageData.DecodeCodeList = append(messageData.DecodeCodeList, d)

			if param.Type == "" {
				continue
			}

			if idx := findByParamList(baseType.ParamList, param); idx == -1 {
				param.Index = len(baseType.ParamList)
				baseType.ParamList = append(baseType.ParamList, param)
			} else {
				param.Index = idx
			}
			messageData.ParamList = append(messageData.ParamList, param)
		}
		baseType.SubMessageList = append(baseType.SubMessageList, messageData)
		// glog.Info(baseType)
		// messages.MessageList = append(messages.MessageList, message)
	}

	types = &TplTypesDataList{}
	for _, v := range baseTypeMap {
		// param.Name有重复，要修改Name
		names := make(map[string][]int)

		for i, p := range v.ParamList {
			v.ParamList[i].Index = i + 1
			if _, ok := names[p.Name]; !ok {
				names[p.Name] = []int{i}
			} else {
				names[p.Name] = append(names[p.Name], i)
			}
		}

		for _, v2 := range names {
			if len(v2) > 1 {
				for _, idx := range v2 {
					v.ParamList[idx].Name = v.ParamList[idx].Name + "_" + strconv.Itoa(v.ParamList[idx].Index)
				}
			}
		}

		for i3, v3 := range v.SubMessageList {
			for i4, v4 := range v3.ParamList {
				// glog.Info(i4, " ==> ", v4, ", ", v3.Line)
				// glog.Info(v.ParamList)
				v4.Name2 = v.ParamList[v4.Index].Name
				v3.ParamList[i4] = v4
			}
			v.SubMessageList[i3] = v3
		}
		// glog.Info(v)
		types.BaseTypeList = append(types.BaseTypeList, *v)
	}

	// glog.Info(types)

	return
}

func makeFunctionDataListTpl(schemas *mtproto_parser.Schemas) (funcs *TplFunctionDataList) {
	// messages := make(map[string]*TplBaseMessage)
	funcs = &TplFunctionDataList{}

	serviceTypeMap := make(map[string]*TplBaseTypeData)

	// RequestList
	for _, c := range schemas.FunctionList {
		rpcName := strings.Split(c.Method, ".")[0]
		// baseMessage := &TplBaseMessage{Name: baseTypeName}
		service, ok := serviceTypeMap[rpcName]
		if !ok {
			service = &TplBaseTypeData{
				Name: rpcName,
			}
			serviceTypeMap[rpcName] = service
		}

		message := TplMessageData{
			Name: strings.Replace(c.Method, ".", "_", -1),
			Line: c.Line,
		}

		serviceMessage := TplMessageData{
			Name: strings.Replace(c.Method, ".", "_", -1),
			Line: c.Line,
		}

		// glog.Info(c.Line)
		for i, p := range c.ParamList {
			param := TplParam{}
			param.Name = p.Name
			param.Index = i + 1
			param.Type = toGolangType(p.Type)
			if param.Type == "" {
				continue
			}
			message.ParamList = append(message.ParamList, param)
		}
		funcs.RequestList = append(funcs.RequestList, message)

		switch c.ResType.(type) {
		case mtproto_parser.TVectorType:
			vectorType := TplParam{
				Type: toGolangType(c.ResType),
				Name: toMessageName(c.ResType.(mtproto_parser.TVectorType).Type.Name()),
			}
			if -1 == findByParamList(funcs.VectorResList, vectorType) {
				funcs.VectorResList = append(funcs.VectorResList, vectorType)
			}
			serviceMessage.ResType = "Vector_" + vectorType.Name
		case mtproto_parser.BuiltInVectorType:
			vectorType := TplParam{
				Type: toGolangType(c.ResType),
				Name: toMessageName(c.ResType.(mtproto_parser.TVectorType).Type.Name()),
			}
			if -1 == findByParamList(funcs.VectorResList, vectorType) {
				funcs.VectorResList = append(funcs.VectorResList, vectorType)
			}
			serviceMessage.ResType = "Vector_" + vectorType.Name
		default:
			serviceMessage.ResType = toMessageName(c.ResType.Name())
		}
		service.SubMessageList = append(service.SubMessageList, serviceMessage)
	}

	for _, v := range serviceTypeMap {
		if checkByStringList(ignoreRpcList, v.Name) {
			continue
		}
		funcs.ServiceList = append(funcs.ServiceList, *v)
		// glog.Info(v)
	}

	return
}
