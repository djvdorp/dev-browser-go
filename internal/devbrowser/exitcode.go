package devbrowser

// ExitCodeError allows cobra handlers to request a specific process exit code.
// main.go should detect this and exit with Code.
type ExitCodeError struct {
	Code int
	Err  error
}

func (e ExitCodeError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "exit"
}

func (e ExitCodeError) Unwrap() error { return e.Err }

func (e ExitCodeError) ExitCode() int {
	if e.Code <= 0 {
		return 1
	}
	return e.Code
}
