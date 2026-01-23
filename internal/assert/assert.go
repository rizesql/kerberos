package assert

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func Equal[T any](tb testing.TB, actual T, expected T) {
	tb.Helper()
	if areEqual(actual, expected) {
		return
	}
	tb.Errorf("actual: %#v; expected: %#v", actual, expected)
}

func Err(tb testing.TB, actual error, expecteds ...any) {
	tb.Helper()

	if len(expecteds) == 0 {
		if actual == nil {
			tb.Error("actual: <nil>; expected: error")
		}
		return
	}

	expected := expecteds[0]

	if expected != nil && actual == nil {
		tb.Error("actual: <nil>; expected: error")
		return
	}

	switch e := expected.(type) {
	case nil:
		if actual != nil {
			tb.Fatalf("unexpected error: %v", actual)
		}
	case string:
		if !strings.Contains(actual.Error(), e) {
			tb.Errorf("actual: %q; expected: %q", actual.Error(), e)
		}
	case error:
		if !errors.Is(actual, e) {
			tb.Errorf("actual: %T(%v); expected: %T(%v)", actual, actual, e, e)
		}
	case reflect.Type:
		target := reflect.New(e).Interface()
		if !errors.As(actual, target) {
			tb.Errorf("actual: %T; expected: %s", actual, e)
		}
	default:
		tb.Errorf("unsupported expected type: %T", expected)
	}
}

func True(tb testing.TB, actual bool) {
	tb.Helper()
	if !actual {
		tb.Error("actual: false; expected: true")
	}
}

type equaler[T any] interface {
	Equal(T) bool
}

func areEqual[T any](lhs, rhs T) bool {
	if isNil(lhs) && isNil(rhs) {
		return true
	}

	if eq, ok := any(lhs).(equaler[T]); ok {
		return eq.Equal(rhs)
	}

	if lhsBytes, ok := any(lhs).([]byte); ok {
		rhsBytes := any(rhs).([]byte)
		return bytes.Equal(lhsBytes, rhsBytes)
	}

	return reflect.DeepEqual(lhs, rhs)
}

func isNil(it any) bool {
	if it == nil {
		return true
	}

	rv := reflect.ValueOf(it)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}
