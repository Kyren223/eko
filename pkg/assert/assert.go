package assert

import "log"

func Assert(assertion bool, message string, a ...any) {
	if !assertion {
		log.Fatalf(message+"\n", a...)
	}
}

func NoError(err error, message string, a ...any) {
	if err != nil {
		log.Fatalf(message+": "+err.Error()+"\n", a...)
	}
}

func Never(message string, a ...any) {
	log.Fatalf(message+"\n", a...)
}

func NotNil(value any, message string, a ...any) {
	if value == nil {
		log.Fatalf(message+"\n", a...)
	}
}
