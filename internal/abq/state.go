package abq

type State struct {
	AbqExecutable string `json:"abq_executable"`
	AbqVersion    string `json:"abq_version"`
	RunID         string `json:"run_id"`
	Supervisor    bool   `json:"supervisor"`
}
