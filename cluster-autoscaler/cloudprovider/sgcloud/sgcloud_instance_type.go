package sgcloud

type InstanceType struct {
	instanceTypeID string
	vcpu           int64
	memoryInBytes  int64
	gpu            int64
}
