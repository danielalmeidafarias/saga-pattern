package saga

import "github.com/danielalmeidafarias/saga-pattern/pkg/msg"

type Status string

const (
	StatusRunning      Status = "RUNNING"
	StatusCompensating Status = "COMPENSATING"
	StatusSucceeded    Status = "SUCCEEDED"
	StatusFailed       Status = "FAILED"
)

type StepStatus string

const (
	StepPending      StepStatus = "PENDING"
	StepDispatched   StepStatus = "DISPATCHED"
	StepSucceeded    StepStatus = "SUCCEEDED"
	StepFailed       StepStatus = "FAILED"
	StepCompensating StepStatus = "COMPENSATING"
	StepCompensated  StepStatus = "COMPENSATED"
)

// Saga is the persisted state of one distributed transaction.
type Saga struct {
	ID       string
	OrderID  string
	Trigger  msg.MessageType
	Status   Status
	StepList []SagaStep
}

// SagaStep stores both the command and its compensation, so execution can be resumed.
type SagaStep struct {
	ID           string
	SagaID       string
	Phase        int
	Status       StepStatus
	Command      msg.Message
	Compensation *msg.Message
}
