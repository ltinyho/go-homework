package hystrix

type event int

const (
	_ event = iota
	success
	failure
	rejected
	shortCircuit
	timeout
	contextCanceled
	contextDeadlineExceeded
	fallbackSuccess
	fallbackFailure
 )
