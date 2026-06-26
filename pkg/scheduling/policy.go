package scheduling

// Decision is the controller output consumed by workflow and Volcano builders.
type Decision struct {
	Queue      string
	GPUProfile string
	MIGProfile string
	Reason     string
}
