package store

type Request interface {}

type ReqRead struct {
	FileName string
}

type ReqWrite struct {
	FileName string
	ExpTime  uint64
	Contents []byte
}

type ReqCaS struct {
	FileName string
	Version  uint64
	ExpTime  uint64
	Contents []byte
}

type ReqDelete struct {
	FileName string
}

type Response interface {}

type ResOk struct{}

type ResContents struct {
	FileName string
	Version  uint64
	ExpTime  uint64
	Contents []byte
}

type ResOkVer struct {
	Version uint64
}

type ResError struct {
	Desc string
}
