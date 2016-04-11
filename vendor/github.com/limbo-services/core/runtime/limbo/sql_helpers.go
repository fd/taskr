package limbo

import (
	"regexp"
	"strings"

	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/generator"
)

func CleanSQL(s string) string {
	s = regexp.MustCompile("\\n+").ReplaceAllString(s, " ")
	s = regexp.MustCompile("\\s+").ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

func IsGoSQLValuer(msg *generator.Descriptor) bool {
	return proto.GetBoolExtension(msg.Options, E_Gosqlvaluer, false)
}

func GetModel(msg *generator.Descriptor) *ModelDescriptor {
	if msg.Options != nil {
		v, _ := proto.GetExtension(msg.Options, E_Model)
		if v != nil {
			return v.(*ModelDescriptor)
		}
	}

	return nil
}

func GetColumn(field *pb.FieldDescriptorProto) *ColumnDescriptor {
	if field.Options == nil {
		field.Options = &pb.FieldOptions{}
	}

	v, _ := proto.GetExtension(field.Options, E_Column)
	if v == nil {
		if j, _ := proto.GetExtension(field.Options, E_Join); j != nil {
			return nil
		}

		v = &ColumnDescriptor{}
		proto.SetExtension(field.Options, E_Column, v)
	}

	c := v.(*ColumnDescriptor)
	if c.Ignore {
		return nil
	}

	return c
}

func GetJoin(field *pb.FieldDescriptorProto) *JoinDescriptor {
	if field.Options != nil {
		v, _ := proto.GetExtension(field.Options, E_Join)
		if v != nil {
			return v.(*JoinDescriptor)
		}
	}

	return nil
}

func (s *ScannerDescriptor) ID() string {
	if s.Name == "" {
		return s.MessageType
	}
	return s.MessageType + ":" + s.Name
}

func (m *ScannerDescriptor) LookupJoin(fieldName string) *JoinDescriptor {
	for _, join := range m.Join {
		if join.FieldName == fieldName {
			return join
		}
	}
	return nil
}

func (m *ScannerDescriptor) LookupColumn(fieldName string) *ColumnDescriptor {
	for _, column := range m.Column {
		if column.FieldName == fieldName {
			return column
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupJoin(fieldName string) *JoinDescriptor {
	for _, join := range m.Join {
		if join.FieldName == fieldName {
			return join
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupColumn(fieldName string) *ColumnDescriptor {
	for _, column := range m.Column {
		if column.FieldName == fieldName {
			return column
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupScanner(fieldName string) *ScannerDescriptor {
	for _, column := range m.Scanner {
		if column.Name == fieldName {
			return column
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupDeepJoin(fieldName string) *JoinDescriptor {
	for _, join := range m.DeepJoin {
		if join.FieldName == fieldName {
			return join
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupDeepColumn(fieldName string) *ColumnDescriptor {
	for _, column := range m.DeepColumn {
		if column.FieldName == fieldName {
			return column
		}
	}
	return nil
}

func (m *ModelDescriptor) LookupDeepScanner(fieldName string) *ScannerDescriptor {
	for _, column := range m.DeepScanner {
		if column.Name == fieldName {
			return column
		}
	}
	return nil
}

type SortedColumnDescriptors []*ColumnDescriptor
type SortedJoinDescriptors []*JoinDescriptor
type SortedScannerDescriptors []*ScannerDescriptor

func (s SortedColumnDescriptors) Len() int  { return len(s) }
func (s SortedJoinDescriptors) Len() int    { return len(s) }
func (s SortedScannerDescriptors) Len() int { return len(s) }

func (s SortedColumnDescriptors) Less(i, j int) bool  { return s[i].FieldName < s[j].FieldName }
func (s SortedJoinDescriptors) Less(i, j int) bool    { return s[i].FieldName < s[j].FieldName }
func (s SortedScannerDescriptors) Less(i, j int) bool { return s[i].Name < s[j].Name }

func (s SortedColumnDescriptors) Swap(i, j int)  { s[i], s[j] = s[j], s[i] }
func (s SortedJoinDescriptors) Swap(i, j int)    { s[i], s[j] = s[j], s[i] }
func (s SortedScannerDescriptors) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
