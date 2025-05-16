package shvars

import (
	"fmt"
	"os"
)

type VariableGetter interface {
	Len() int

	Get(varName string) (value string, hasIt bool)
}

func Replace(args []string, vars VariableGetter) error {
	if vars.Len() == 0 {
		return nil
	}

	var err error

	replaceFn := func(varName string) string {
		if err != nil {
			return varName
		}

		replacement, hasIt := vars.Get(varName)
		if !hasIt {
			err = fmt.Errorf("unknown variable: %q", varName)

			return varName
		}

		return replacement
	}

	for i, arg := range args {
		result := os.Expand(arg, replaceFn)
		if arg != result {
			args[i] = result
		}
	}

	if err != nil {
		return err
	}

	return nil
}
