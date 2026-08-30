package biz

const protoErrorReasonPrefix = "ERROR_REASON_"

type protoErrorReason interface {
	String() string
}

func reasonFromProto(reason protoErrorReason) string {
	return reason.String()[len(protoErrorReasonPrefix):]
}
