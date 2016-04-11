package limbo

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
)

var definitions = map[string]SchemaDefinition{}
var operations = map[string]map[string]SwaggerOperation{}

type SchemaDefinition struct {
	Name         string
	Definition   []byte
	Dependencies []string
}

type SwaggerOperation struct {
	Pattern      string
	Method       string
	Definition   []byte
	Dependencies []string
}

func RegisterSchemaDefinitions(defs []SchemaDefinition) {
	var buf bytes.Buffer
	for _, def := range defs {
		buf.Reset()
		if err := json.Compact(&buf, def.Definition); err == nil {
			def.Definition = append(def.Definition[:0], buf.Bytes()...)
		}
		definitions[def.Name] = def
	}
}

func RegisterSwaggerOperations(defs []SwaggerOperation) {
	var buf bytes.Buffer
	for _, def := range defs {
		buf.Reset()
		if err := json.Compact(&buf, def.Definition); err == nil {
			def.Definition = append(def.Definition[:0], buf.Bytes()...)
		}

		m := operations[def.Pattern]
		if m == nil {
			m = map[string]SwaggerOperation{}
			operations[def.Pattern] = m
		}

		m[strings.ToLower(def.Method)] = def
	}
}

func IsRequiredProperty(field *pb.FieldDescriptorProto) bool {
	if field.Options != nil && proto.HasExtension(field.Options, E_Required) {
		return proto.GetBoolExtension(field.Options, E_Required, false)
	}
	if field.IsRequired() {
		return true
	}
	return false
}

func GetFormat(field *pb.FieldDescriptorProto) (string, bool) {
	if field == nil || field.Options == nil {
		return "", false
	}
	v, _ := proto.GetExtension(field.Options, E_Format)
	s, _ := v.(*string)
	if s == nil {
		return "", false
	}
	return *s, true
}

func GetPattern(field *pb.FieldDescriptorProto) (string, bool) {
	if field == nil || field.Options == nil {
		return "", false
	}
	v, _ := proto.GetExtension(field.Options, E_Pattern)
	s, _ := v.(*string)
	if s == nil {
		return "", false
	}
	return *s, true
}

func GetMinLength(field *pb.FieldDescriptorProto) (uint32, bool) {
	if field == nil || field.Options == nil {
		return 0, false
	}
	v, _ := proto.GetExtension(field.Options, E_MinLength)
	s, _ := v.(*uint32)
	if s == nil {
		return 0, false
	}
	return *s, true
}

func GetMaxLength(field *pb.FieldDescriptorProto) (uint32, bool) {
	if field == nil || field.Options == nil {
		return 0, false
	}
	v, _ := proto.GetExtension(field.Options, E_MaxLength)
	s, _ := v.(*uint32)
	if s == nil {
		return 0, false
	}
	return *s, true
}

func GetMinItems(field *pb.FieldDescriptorProto) (uint32, bool) {
	if field == nil || field.Options == nil {
		return 0, false
	}
	v, _ := proto.GetExtension(field.Options, E_MinItems)
	s, _ := v.(*uint32)
	if s == nil {
		return 0, false
	}
	return *s, true
}

func GetMaxItems(field *pb.FieldDescriptorProto) (uint32, bool) {
	if field == nil || field.Options == nil {
		return 0, false
	}
	v, _ := proto.GetExtension(field.Options, E_MaxItems)
	s, _ := v.(*uint32)
	if s == nil {
		return 0, false
	}
	return *s, true
}

type SwaggerInfo struct {
	Host           string
	Schemes        []string
	Title          string
	Description    string
	TermsOfService string
	Version        string
}

type swaggerInfo struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Version        string `json:"version"`
}

type swaggerRoot struct {
	Swagger     string                                 `json:"swagger"`
	Info        swaggerInfo                            `json:"info"`
	Host        string                                 `json:"host"`
	Schemes     []string                               `json:"schemes"`  // ["https", "wss"]
	Consumes    []string                               `json:"consumes"` // ["application/json; charset=utf-8"]
	Produces    []string                               `json:"produces"` // ["application/json; charset=utf-8"]
	Paths       map[string]map[string]*json.RawMessage `json:"paths"`
	Definitions map[string]*json.RawMessage            `json:"definitions"`
	// SecurityDefinitions map[string]*SwaggerSecurityDefinition `json:"securityDefinitions"`
}

func MakeSwaggerSpec(info *SwaggerInfo) ([]byte, error) {
	root := swaggerRoot{
		Swagger:  "2.0",
		Host:     info.Host,
		Schemes:  info.Schemes,
		Consumes: []string{"application/json"},
		Produces: []string{"application/json"},
		Info: swaggerInfo{
			Title:          info.Title,
			Description:    info.Description,
			TermsOfService: info.TermsOfService,
			Version:        info.Version,
		},
		Definitions: map[string]*json.RawMessage{},
		Paths:       map[string]map[string]*json.RawMessage{},
	}

	for pattern, x := range operations {
		y := map[string]*json.RawMessage{}
		root.Paths[pattern] = y
		for method, decl := range x {

			data := decl.Definition
			for _, dep := range decl.Dependencies {
				data = bytes.Replace(data, []byte("\""+dep+"\""), []byte("\"#/definitions/"+dep+"\""), -1)
			}

			y[method] = (*json.RawMessage)(&data)

			err := addDependencies(root.Definitions, decl.Dependencies)
			if err != nil {
				return nil, err
			}
		}
	}

	return json.Marshal(root)
}

func addDependencies(defs map[string]*json.RawMessage, deps []string) error {
	for _, dep := range deps {
		if _, ok := defs[dep]; ok {
			continue
		}

		decl, ok := definitions[dep]
		if !ok {
			continue
		}

		data := decl.Definition
		for _, dep := range decl.Dependencies {
			data = bytes.Replace(data, []byte("\""+dep+"\""), []byte("\"#/definitions/"+dep+"\""), -1)
		}

		defs[dep] = (*json.RawMessage)(&data)
		err := addDependencies(defs, decl.Dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}
