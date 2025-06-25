package bridge

type StdioToSSE struct {
	command string
	args    []string
}

func NewStdioToSSE() Bridge {
	return &StdioToSSE{}
}

func (b *StdioToSSE) Start() error {
	return nil
}

func (b *StdioToSSE) Stop() error {
	return nil
}

func (b *StdioToSSE) Restart() error {
	return nil
}

func (b *StdioToSSE) IsRunning() bool {
	return false
}

func (b *StdioToSSE) GetURL() string {
	return ""
}

func (b *StdioToSSE) GetSSEURL() string {
	return ""
}

func (b *StdioToSSE) GetMessageURL() string {
	return ""
}
