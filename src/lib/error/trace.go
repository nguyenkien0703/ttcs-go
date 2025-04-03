package lib_error

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

// エラー時のダンプ出力用の構造体.
type CallerInfo struct {
	PackageName  string
	FunctionName string
	FileName     string
	FileLine     int
}

// エラー時のダンプ出力用関数.
func dump() []*CallerInfo {
	callerInfos := []*CallerInfo{}
	re := regexp.MustCompile(`^(\S.+)\.(\S.+)$`)
	for i := 1; ; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		fileName, fileLine := fn.FileLine(pc)
		_fn := re.FindStringSubmatch(fn.Name())
		packageName := ""
		functionName := ""
		if len(_fn) > 0 {
			packageName = _fn[1]
			if len(_fn) > 1 {
				functionName = _fn[2]
			}
		}
		callerInfos = append(callerInfos, &CallerInfo{
			PackageName:  packageName,
			FunctionName: functionName,
			FileName:     fileName,
			FileLine:     fileLine,
		})
	}
	return callerInfos
}

// 現在のスタックトレース.
func StackTrace() string {
	infos := dump()
	length := len(infos)
	messages := make([]string, length)
	for i, v := range infos {
		messages[length-i-1] = fmt.Sprintf("%02d: %s.%s@%s:%d", i, v.PackageName, v.FunctionName, v.FileName, v.FileLine)
	}
	return strings.Join(messages, "\n")
}
