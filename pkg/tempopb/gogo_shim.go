package tempopb

import "github.com/gogo/protobuf/proto"

// Register enums with gogo/protobuf so jsonpb (and any other consumer that
// goes through proto.EnumValueMap) can look up the symbolic names that
// wiresmith only exposes via the local *_value/*_name maps. Mirrors the
// *_gogo_shim.go files in pkg/tempopb/{common,trace}/v1.

func init() {
	proto.RegisterEnum("tempopb.PushErrorReason", PushErrorReason_name, PushErrorReason_value)
	proto.RegisterEnum("tempopb.PartialStatus", PartialStatus_name, PartialStatus_value)
	proto.RegisterEnum("tempopb.DedicatedColumn.Scope", DedicatedColumn_Scope_name, DedicatedColumn_Scope_value)
	proto.RegisterEnum("tempopb.DedicatedColumn.Type", DedicatedColumn_Type_name, DedicatedColumn_Type_value)
	proto.RegisterEnum("tempopb.DedicatedColumn.Option", DedicatedColumn_Option_name, DedicatedColumn_Option_value)
	proto.RegisterEnum("tempopb.JobType", JobType_name, JobType_value)
	proto.RegisterEnum("tempopb.JobStatus", JobStatus_name, JobStatus_value)
}
