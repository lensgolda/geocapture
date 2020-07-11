package interfaces

type LogFailer interface {
	LogFailed(recordID int)
}
