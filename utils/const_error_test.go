package utils

import (
	"errors"
	"fmt"
	"testing"
)

func TestConstError(t *testing.T) {
	const errConst = ConstError("const")
	var _ error = errConst
	err := fmt.Errorf("wrap: %w", errConst)
	if !errors.Is(err, errConst) {
		t.Errorf("expect err is errConst, but not")
	}
}
