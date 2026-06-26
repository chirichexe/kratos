package scheduling

// Decision is the controller output consumed by Volcano builders.
type Decision struct {
	Queue        string
	SelectedNode string
	NodeSelector map[string]string
	Reason       string
}
