package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fmt"
	"html/template"
	"runtime/debug"

	"github.com/fatih/color"
)

func strftime(format string, timestamp interface{}) string {
	format = strings.ReplaceAll(format, "%Y", "2006")
	format = strings.ReplaceAll(format, "%m", "01")
	format = strings.ReplaceAll(format, "%d", "02")
	format = strings.ReplaceAll(format, "%H", "15")
	format = strings.ReplaceAll(format, "%M", "04")
	format = strings.ReplaceAll(format, "%S", "05")
	return time.Unix(toInt64(timestamp), 0).Format(format)
}

func sleep(t interface{}) {
	time.Sleep(getTimeDuration(t))
}

func strIndex(str, substr string) int {
	pos := strings.Index(str, substr)
	return pos
}

func strReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func strJoin(glue string, pieces []string) string {
	var buf bytes.Buffer
	l := len(pieces)
	for _, str := range pieces {
		buf.WriteString(str)
		if l--; l > 0 {
			buf.WriteString(glue)
		}
	}
	return buf.String()
}

func strStrip(str string, characterMask ...string) string {
	if len(characterMask) == 0 {
		return strings.TrimSpace(str)
	}
	return strings.Trim(str, characterMask[0])
}

func base64Decode(str string) string {
	switch len(str) % 4 {
	case 2:
		str += "=="
	case 3:
		str += "="
	}

	data, err := base64.StdEncoding.DecodeString(str)
	panicerr(err)
	return string(data)
}

func pathExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func unlink(filename string) {
	err := os.RemoveAll(filename)
	panicerr(err)
}

func basename(path string) string {
	return filepath.Base(path)
}

func system(command string, timeoutSecond ...interface{}) int {
	q := rune(0)
	parts := strings.FieldsFunc(command, func(r rune) bool {
		switch {
		case r == q:
			q = rune(0)
			return false
		case q != rune(0):
			return false
		case unicode.In(r, unicode.Quotation_Mark):
			q = r
			return false
		default:
			return unicode.IsSpace(r)
		}
	})
	// remove the " and ' on both sides
	for i, v := range parts {
		f, l := v[0], len(v)
		if l >= 2 && (f == '"' || f == '\'') {
			parts[i] = v[1 : l-1]
		}
	}

	if !cmdExists(parts[0]) {
		panicerr("Command not exists")
	}

	var statuscode int
	if len(timeoutSecond) != 0 {
		t := timeoutSecond[0]
		timeoutDuration := getTimeDuration(t)
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command) // 如果不是bash会kill不掉
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			e := err.(*exec.ExitError)
			statuscode = e.ExitCode()
		} else {
			statuscode = 0
		}
	} else {
		// cmd := exec.Command("/bin/bash", "-c", command)
		cmd := exec.Command(parts[0], parts[1:]...)

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			e := err.(*exec.ExitError)
			statuscode = e.ExitCode()
		} else {
			statuscode = 0
		}
	}
	return statuscode
}

func envexists(varname string) bool {
	_, exist := os.LookupEnv(varname)
	return exist
}

func getenv(varname string) string {
	e, exist := os.LookupEnv(varname)
	if !exist {
		err := errors.New("环境变量不存在")
		panicerr(err)
	}
	return e
}

func reFindAll(pattern string, text string, multiline ...bool) [][]string {
	if len(multiline) > 0 && multiline[0] {
		pattern = "(?s)" + pattern
	}
	r, err := regexp.Compile(pattern)
	panicerr(err)
	return r.FindAllStringSubmatch(text, -1)
}

func strSplit(str string, sep ...string) []string {
	var a []string
	if len(sep) != 0 {
		for _, v := range strings.Split(str, sep[0]) {
			a = append(a, strStrip(v))
		}
	} else {
		for _, v := range strings.Split(str, " ") {
			if strStrip(v) != "" {
				a = append(a, strStrip(v))
			}
		}
	}

	return a
}

func itemInArray(item interface{}, array interface{}) bool {
	// 获取值的列表
	arr := reflect.ValueOf(array)

	// 手工判断值的类型
	if arr.Kind() != reflect.Array && arr.Kind() != reflect.Slice {
		panicerr("Invalid data type of param \"array\": Not an Array")
	}

	// 遍历值的列表
	for i := 0; i < arr.Len(); i++ {
		// 取出值列表的元素并转换为interface
		if arr.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

type jsonMap map[string]interface{}
type jsonArr []interface{}

func jsonDumps(v interface{}, pretty ...bool) string {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	if len(pretty) != 0 {
		encoder.SetIndent(" ", " ")
	}
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)

	panicerr(err)
	return strStrip(buffer.String())
}

func strIn(substr string, str string) bool {
	if strIndex(str, substr) != -1 {
		return true
	}
	return false
}

type lockStruct struct {
	lock *sync.Mutex
}

func getLock() *lockStruct {
	var a sync.Mutex
	return &lockStruct{lock: &a}
}

func (m *lockStruct) acquire() {
	m.lock.Lock()
}

func (m *lockStruct) release() {
	m.lock.Unlock()
}

func strStartsWith(str string, substr string) (res bool) {
	if len(substr) <= len(str) && str[:len(substr)] == substr {
		res = true
	} else {
		res = false
	}
	return
}
func pathJoin(args ...string) string {
	return path.Join(args...)
}

var lg *logStruct

func init() {
	lg = getLogger()
	// os.Setenv("TZ", "Asia/Hong_Kong")
}

type exception struct {
	Error error
}

type tryConfig struct {
	retry int // retry times while error occure
	sleep int // sleep seconds between retry
}

func try(f func(), trycfg ...tryConfig) (e exception) {
	if len(trycfg) == 0 {
		e = exception{nil}
		defer func() {
			if err := recover(); err != nil {
				errmsg := fmt.Sprintf("%s", err)
				if len(reFindAll(".+\\.go:[0-9]+ >> .+? >> \\(.+?\\)", errmsg)) == 0 {
					e.Error = errors.New(fmtDebugStack(errmsg, str(debug.Stack())))
				} else {
					e.Error = errors.New(errmsg)
				}
			}
		}()
		f()
		return
	}
	for i := 0; ; i++ {
		e = func() (e exception) {
			e = exception{nil}
			defer func() {
				if err := recover(); err != nil {
					e.Error = errors.New(fmt.Sprintf("%s", err))
				}
			}()
			f()
			return
		}()
		if e.Error == nil {
			return
		}
		if e.Error != nil && trycfg[0].retry > 0 && i >= trycfg[0].retry {
			break
		}
		sleep(trycfg[0].sleep)
	}
	return
}

type errorStruct struct {
	msg string
}

func fmtDebugStack(msg string, stack string) string {
	//lg.debug("msg:", msg)
	//lg.debug("stack:", stack)

	blackFileList := []string{
		"lib.go",
		"stack.go",
	}

	l := reFindAll("([\\-a-zA-Z0-9]+\\.go:[0-9]+)", stack)
	//lg.debug(l)
	for i, j := 0, len(l)-1; i < j; i, j = i+1, j-1 {
		l[i], l[j] = l[j], l[i]
	}
	//lg.debug(l)

	var link []string
	for _, f := range l {
		ff := strings.Split(f[0], ":")[0]
		inside := func(a string, list []string) bool {
			for _, b := range list {
				if b == a {
					return true
				}
			}
			return false
		}(ff, blackFileList)
		if !inside {
			link = append(link, f[0])
		}
	}
	//lg.debug(link)

	var strr string
	if len(link) != 1 {
		// strr = link[len(link)-2] + " >> " + msg + " >> " + "(" + strJoin(" => ", link[:len(link)-1]) + ")"
		strr = link[len(link)-1] + " >> " + msg + " >> " + "(" + strJoin(" => ", link) + ")"
	} else {
		strr = link[0] + " >> " + msg
	}

	//lg.debug("strr:", strr)
	return strr
}

func panicerr(err interface{}) {
	switch t := err.(type) {
	case string:
		// lg.trace("1")
		panic(fmtDebugStack(t, string(debug.Stack())))
	case error:
		// lg.trace("2")
		panic(fmtDebugStack(t.Error(), string(debug.Stack())))
	case *errorStruct:
		// lg.trace(3)
		panic(t.msg)
	case nil:
		return
	default:
		panic(fmtDebugStack(fmt.Sprintf("%s", t), string(debug.Stack())))
	}
}

func toFloat64E(i interface{}) (float64, error) {
	i = indirect(i)

	switch s := i.(type) {
	case float64:
		return s, nil
	case float32:
		return float64(s), nil
	case int:
		return float64(s), nil
	case int64:
		return float64(s), nil
	case int32:
		return float64(s), nil
	case int16:
		return float64(s), nil
	case int8:
		return float64(s), nil
	case uint:
		return float64(s), nil
	case uint64:
		return float64(s), nil
	case uint32:
		return float64(s), nil
	case uint16:
		return float64(s), nil
	case uint8:
		return float64(s), nil
	case string:
		v, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return v, nil
		}
		return 0, fmt.Errorf("unable to cast %#v of type %T to float64", i, i)
	case bool:
		if s {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unable to cast %#v of type %T to float64", i, i)
	}
}

func toInt64E(i interface{}) (int64, error) {
	i = indirect(i)

	switch s := i.(type) {
	case int:
		return int64(s), nil
	case int64:
		return s, nil
	case int32:
		return int64(s), nil
	case int16:
		return int64(s), nil
	case int8:
		return int64(s), nil
	case uint:
		return int64(s), nil
	case uint64:
		return int64(s), nil
	case uint32:
		return int64(s), nil
	case uint16:
		return int64(s), nil
	case uint8:
		return int64(s), nil
	case float64:
		return int64(s), nil
	case float32:
		return int64(s), nil
	case string:
		v, err := strconv.ParseInt(s, 0, 0)
		if err == nil {
			return v, nil
		}
		return 0, fmt.Errorf("unable to cast %#v of type %T to int64", i, i)
	case bool:
		if s {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unable to cast %#v of type %T to int64", i, i)
	}
}

func indirect(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

func indirecttoStringerOrError(a interface{}) interface{} {
	if a == nil {
		return nil
	}

	var errorType = reflect.TypeOf((*error)(nil)).Elem()
	var fmtStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

	v := reflect.ValueOf(a)
	for !v.Type().Implements(fmtStringerType) && !v.Type().Implements(errorType) && v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

func toStringE(i interface{}) (string, error) {
	i = indirecttoStringerOrError(i)

	switch s := i.(type) {
	case string:
		return s, nil
	case bool:
		return strconv.FormatBool(s), nil
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(s), 'f', -1, 32), nil
	case int:
		return strconv.Itoa(s), nil
	case int64:
		return strconv.FormatInt(s, 10), nil
	case int32:
		return strconv.Itoa(int(s)), nil
	case int16:
		return strconv.FormatInt(int64(s), 10), nil
	case int8:
		return strconv.FormatInt(int64(s), 10), nil
	case uint:
		return strconv.FormatUint(uint64(s), 10), nil
	case uint64:
		return strconv.FormatUint(uint64(s), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(s), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(s), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(s), 10), nil
	case []byte:
		return string(s), nil
	case template.HTML:
		return string(s), nil
	case template.URL:
		return string(s), nil
	case template.JS:
		return string(s), nil
	case template.CSS:
		return string(s), nil
	case template.HTMLAttr:
		return string(s), nil
	case nil:
		return "", nil
	case fmt.Stringer:
		return s.String(), nil
	case error:
		return s.Error(), nil
	default:
		return "", fmt.Errorf("unable to cast %#v of type %T to string", i, i)
	}
}

func toFloat64(i interface{}) float64 {
	v, err := toFloat64E(i)
	panicerr(err)
	return v
}

func toInt64(i interface{}) int64 {
	v, err := toInt64E(i)
	panicerr(err)
	return v
}

func toString(i interface{}) string {
	v, err := toStringE(i)
	panicerr(err)
	return v
}

func str(i interface{}) string {
	v, err := toStringE(i)
	panicerr(err)
	return v
}

func typeof(v interface{}) string {
	return reflect.TypeOf(v).String()
}

func now() float64 {
	return toFloat64(time.Now().UnixMicro()) / 1000000
}

func getTimeDuration(t interface{}) time.Duration {
	var timeDuration time.Duration
	if typeof(t) == "float64" {
		tt := t.(float64) * 1000
		if tt < 0 {
			tt = 0
		}
		timeDuration = time.Duration(tt) * time.Millisecond
	}
	if typeof(t) == "int" || typeof(t) == "int8" || typeof(t) == "int16" || typeof(t) == "int32" || typeof(t) == "int64" {
		tt := toInt64(t)
		if tt < 0 {
			tt = 0
		}
		timeDuration = time.Duration(tt) * time.Second
	}
	return timeDuration
}

func getGoroutineID() int64 {
	var (
		buf [64]byte
		n   = runtime.Stack(buf[:], false)
		stk = strings.TrimPrefix(string(buf[:n]), "goroutine ")
	)

	idField := strings.Fields(stk)[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Errorf("can not get goroutine id: %v", err))
	}

	return int64(id)
}

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

type logStruct struct {
	level                    []string
	json                     bool
	color                    bool
	logDir                   string
	logFileName              string
	logFileSuffix            string
	fd                       *fileStruct
	displayOnTerminal        bool
	lock                     *lockStruct
	logfiles                 []string
	maxlogfiles              int
	logFileSizeInMB          int
	currentLogFileSizeInByte int
	currentLogFileNumber     int
}

func getLogger() *logStruct {
	return &logStruct{
		level:                    []string{"TRAC", "DEBU", "INFO", "WARN", "ERRO"},
		color:                    true,
		displayOnTerminal:        true,
		lock:                     getLock(),
		logFileSizeInMB:          0,
		currentLogFileSizeInByte: 0,
		currentLogFileNumber:     0,
	}
}

func (m *logStruct) trace(args ...interface{}) {
	t := strftime("%m-%d %H:%M:%S", now())
	level := "TRAC"

	var msgarr []string
	for _, a := range args {
		msgarr = append(msgarr, fmt.Sprint(a))
	}
	msg := strJoin(" ", msgarr)

	_, file, no, _ := runtime.Caller(1)
	position := basename(file) + ":" + toString(no)

	m.show(t, level, msg, position)
}

func (m *logStruct) show(t string, level string, msg string, position string) {
	if itemInArray(level, m.level) {
		var strMsg string
		if m.json {
			strMsg = jsonDumps(map[string]string{
				"time":    t,
				"level":   level,
				"message": msg,
			})
		} else {
			msg = strReplace(msg, "\n", "\n                      ")
			if m.color {
				if level == "ERRO" {
					strMsg = color.RedString(t + fmt.Sprintf(" %3v", getGoroutineID()) + " [" + level + "] (" + position + ") " + msg)
				} else if level == "WARN" {
					strMsg = color.YellowString(t + fmt.Sprintf(" %3v", getGoroutineID()) + " [" + level + "] (" + position + ") " + msg)
				} else if level == "INFO" {
					strMsg = color.HiBlueString(t + fmt.Sprintf(" %3v", getGoroutineID()) + " [" + level + "] (" + position + ") " + msg)
				} else if level == "TRAC" {
					strMsg = color.MagentaString(t + fmt.Sprintf(" %3v", getGoroutineID()) + " [" + level + "] (" + position + ") " + msg)
				} else if level == "DEBU" {
					strMsg = color.HiCyanString(t + fmt.Sprintf(" %3v", getGoroutineID()) + " [" + level + "] (" + position + ") " + msg)
				}
			} else {
				strMsg = t + "[" + level + "] (" + position + ") " + msg
			}
		}

		m.lock.acquire()
		if m.displayOnTerminal {
			fmt.Println(strMsg)
		}
		if m.fd != nil {
			if m.logFileSizeInMB == 0 {
				if m.fd.path != pathJoin(m.logDir, m.logFileName+"."+strftime("%Y-%m-%d", now())+"."+m.logFileSuffix) {
					m.fd.close()
					logpath := pathJoin(m.logDir, m.logFileName+"."+strftime("%Y-%m-%d", now())+"."+m.logFileSuffix)
					m.fd = open(logpath, "a")
					m.logfiles = append(m.logfiles, logpath)
					if len(m.logfiles) > m.maxlogfiles {
						unlink(m.logfiles[0])
						m.logfiles = m.logfiles[1:]
					}
				}
			} else {
				if m.currentLogFileSizeInByte > m.logFileSizeInMB*1024*1024 {
					m.currentLogFileSizeInByte = 0
					m.fd.close()
					var logpath string
					for {
						logpath = pathJoin(m.logDir, m.logFileName+"."+str(m.currentLogFileNumber)+"."+m.logFileSuffix)
						if pathExists(logpath) {
							m.currentLogFileNumber++
						} else {
							break
						}
					}
					m.fd = open(logpath, "a")
					m.logfiles = append(m.logfiles, logpath)
					if len(m.logfiles) > m.maxlogfiles {
						unlink(m.logfiles[0])
						m.logfiles = m.logfiles[1:]
					}
				}
			}
			m.fd.write(strMsg + "\n")
			m.currentLogFileSizeInByte = m.currentLogFileSizeInByte + len(strMsg) + 1
		}
		m.lock.release()
	}
}

type fileStruct struct {
	path string
	fd   *os.File
	mode string
	lock *lockStruct
}

func (m *fileStruct) close() {
	m.fd.Close()
}

func (m *fileStruct) write(str interface{}) *fileStruct {
	m.lock.acquire()
	defer m.lock.release()
	var buf []byte
	if typeof(str) == "string" {
		s := str.(string)
		buf = []byte(s)
	} else {
		s := str.([]byte)
		buf = s
	}
	m.fd.Write(buf)
	return m
}

func open(args ...string) *fileStruct {
	path := args[0]
	var mode string
	if len(args) != 1 {
		mode = args[1]
	} else {
		mode = "r"
	}
	var fd *os.File
	var err error
	if string(mode[0]) == "r" {
		fd, err = os.Open(path)
	}
	if string(mode[0]) == "a" {
		fd, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if string(mode[0]) == "w" {
		fd, err = os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	}
	panicerr(err)
	return &fileStruct{
		path: path,
		fd:   fd,
		mode: mode,
		lock: getLock(),
	}
}
