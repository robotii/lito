package vm

import (
	"fmt"
	"strings"
)

func (cf *goCallFrame) inspect() string {
	return fmt.Sprintf("Go method frame. File name: %s. Method name: %s.", cf.FileName(), cf.name)
}

func (cf *CallFrame) inspect() string {
	if cf.ep != nil {
		return fmt.Sprintf("Normal frame. File name: %s. IS name: %s. is block: %t. ep: %d. source line: %d", cf.FileName(), cf.instructionSet.Name, cf.isBlock, len(cf.ep.locals), cf.SourceLine())
	}
	return fmt.Sprintf("Normal frame. File name: %s. IS name: %s. is block: %t. source line: %d", cf.FileName(), cf.instructionSet.Name, cf.isBlock, cf.SourceLine())
}

func (cfs *callFrameStack) inspect() string {
	var out strings.Builder

	for _, cf := range cfs.callFrames {
		if cf != nil {
			out.WriteString(fmt.Sprintln(cf.inspect()))
		}
	}

	return out.String()
}

func (s *Stack) inspect() string {
	var out strings.Builder
	var datas []string

	for i, p := range s.data {
		if p != nil {
			o := p
			if i == s.pointer {
				datas = append(datas, fmt.Sprintf("%s (%T) %d <----", o.ToString(nil), o, i))
			} else {
				datas = append(datas, fmt.Sprintf("%s (%T) %d", o.ToString(nil), o, i))
			}

		} else {
			if i == s.pointer {
				datas = append(datas, "nil <----")
			} else {
				datas = append(datas, "nil")
			}

		}

	}

	out.WriteString("-----------\n")
	out.WriteString(strings.Join(datas, "\n"))
	out.WriteString("\n---------\n")

	return out.String()
}
