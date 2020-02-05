package zenoss

import structpb "github.com/golang/protobuf/ptypes/struct"

func copyMap(m map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func metadataFieldsFromMap(m map[string]string) *structpb.Struct {
	fields := map[string]*structpb.Value{}

	for k, v := range m {
		if k == ImpactFromDimensionsField || k == ImpactToDimensionsField {
			fields[k] = valueFromStringSlice([]string{v})
		} else {
			fields[k] = valueFromString(v)
		}
	}

	return &structpb.Struct{Fields: fields}
}

func valueFromString(s string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: s,
		},
	}
}

func valueFromStringSlice(ss []string) *structpb.Value {
	stringValues := make([]*structpb.Value, len(ss))
	for i, s := range ss {
		stringValues[i] = valueFromString(s)
	}
	return &structpb.Value{
		Kind: &structpb.Value_ListValue{
			ListValue: &structpb.ListValue{
				Values: stringValues,
			},
		},
	}
}
