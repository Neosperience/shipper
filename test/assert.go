package test

type testLogger interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

// AssertExpected checks that a value is as expected and fails the test if it is not.
func AssertExpected[T comparable](t testLogger, val T, expected T, errorMessage string) {
	if val != expected {
		t.Fatalf("%s [expected=\"%v\" | got=\"%v\"]", errorMessage, expected, val)
	}
}

// MustSucceed checks that an error is nil and fails the test otherwise
func MustSucceed(t testLogger, err error, errorMessage string) {
	if err != nil {
		t.Fatalf("%s [error=\"%v\"]", errorMessage, err)
	}
}

// MustFail checks that an error is not nil and fails the test otherwise
func MustFail(t testLogger, err error, errorMessage string) {
	if err == nil {
		t.Fatal(errorMessage)
	}
}
