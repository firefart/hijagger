package main

type WhoisError struct {
	repeatedError bool
	err           error
}

func (err *WhoisError) Error() string {
	return err.err.Error()
}
