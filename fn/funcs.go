package fn

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

func Sum(b []byte, hash hash.Hash) []byte {
	hash.Write(b)
	return hash.Sum(nil)
}

func Sha1Sum(b []byte) string {
	return hex.EncodeToString(Sum(b, sha1.New()))
}

// Sha256 计算值
func Sha256sum(b []byte) string {
	return hex.EncodeToString(Sum(b, sha256.New()))
}

// Md5 计算值
func Md5sum(b []byte) string {
	return hex.EncodeToString(Sum(b, md5.New()))
}

// IsBlank 判断值是否为空
func IsBlank(v any) bool {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	case reflect.Slice:
		return value.Len() == 0
	}
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

// Exist 判断路径是否存在
func Exist(p string) bool {
	_, err := os.Stat(p)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func SliceTurning[T comparable](target []T, limit int) chan []T {
	var (
		total = len(target)
		value = make(chan []T)
	)

	go func() {
		if limit > 0 {
			for offset := 0; offset < total; offset += limit {
				if offset+limit > total {
					limit = total % limit
				}
				value <- target[offset : offset+limit]
			}
		} else {
			value <- target
		}

		close(value)
	}()

	return value
}

func ToString(b []byte) string {
	//return *(*string)(unsafe.Pointer(&b))
	return unsafe.String(&b[0], len(b))
}

func ToBytes(s string) []byte {
	//strHeader := (*[2]uintptr)(unsafe.Pointer(&s))
	//sliceHeader := [3]uintptr{strHeader[0], strHeader[1], strHeader[1]}
	//return *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// TestPrintJson 以json格式打印测试结果
func PrintJson(value interface{}) (e error) {
	v, ok := value.([]byte)
	if !ok {
		v, e = json.Marshal(value)
		if e != nil {
			return e
		}
	}
	var prettyJSON bytes.Buffer
	if e := json.Indent(&prettyJSON, v, "", "\t"); e != nil {
		return e
	}

	fmt.Println(prettyJSON.String())
	return
}

// WriteJson 以json格式写入某个文件路径
func WriteJson(value interface{}, file string) error {
	b, _ := json.Marshal(value)
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, e := fd.Write(b)
	return e
}

// InSlice 判断是否在切片里
func InSlice[T comparable](target []T) func(T) bool {
	set := make(map[T]struct{})
	for _, e := range target {
		set[e] = struct{}{}
	}
	return func(s T) bool {
		_, ok := set[s]
		return ok
	}
}

// Set 切片去重复
func Set[T comparable](s []T) (r []T) {
	var tmp = make(map[T]struct{})
	for i := 0; i < len(s); i++ {
		tmp[s[i]] = struct{}{}
	}
	for key := range tmp {
		r = append(r, key)
	}
	return
}

// LastCut
func LastCut(src, sep string) (before, after string, found bool) {
	idx := strings.LastIndex(src, sep)
	if idx < 0 {
		return src, "", false
	}
	return src[:idx], src[idx+len(sep):], true
}

func zipWriter(p string, info os.FileInfo, archive *zip.Writer) error {
	reader, readErr := os.Open(p)
	if readErr != nil {
		return readErr
	}
	fh := &zip.FileHeader{
		Name:     p,
		Method:   zip.Deflate,
		Modified: info.ModTime(),
	}
	writer, headerErr := archive.CreateHeader(fh)
	//writer, headerErr := archive.Create(p)
	if headerErr != nil {
		return headerErr
	}
	_, err := io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return reader.Close()
}

// ZipFile 压缩某个路径的文件或者文件夹，生成一个target的zip文件
func ZipFile(source, target string, level int) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	if filepath.IsAbs(source) {
		if err := os.Chdir(filepath.Dir(source)); err != nil {
			return err
		}
		source = filepath.Base(source)
	}
	defer zipFile.Close()
	archive := zip.NewWriter(zipFile)
	archive.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, level)
	})
	defer archive.Close()
	stat, statErr := os.Stat(source)
	if statErr != nil {
		return statErr
	}
	if stat.IsDir() {
		return filepath.Walk(source, func(p string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return zipWriter(p, info, archive)
			}
			return nil
		})
	}
	return zipWriter(source, stat, archive)
}

// 读取文件前 n行
func Head(f string, n int) string {
	fd, e := os.Open(f)
	if e != nil {
		panic(e)
	}
	defer fd.Close()
	var result strings.Builder
	buf := bufio.NewScanner(fd)
	for buf.Scan() && n > 0 {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(buf.Text())
		n--
	}
	return result.String()
}

// Tail 输出文件 第n个
func Tail(f string, n int) string {
	var (
		buffSize int64 = 128
		buff           = make([]byte, buffSize)
		now      int64
		ee       error
	)
	fd, e := os.Open(f)
	if e != nil {
		panic(e)
	}
	defer fd.Close()
	// 移动指针到文件末尾
	now, ee = fd.Seek(0, io.SeekEnd)
	if ee != nil {
		panic(ee)
	}

	if n == 0 || now == 0 {
		return ""
	} else if now <= buffSize {
		now, ee = fd.Seek(0, io.SeekStart)
		if ee != nil {
			panic(ee)
		}
		readSize, readErr := fd.Read(buff)
		if readErr != nil {
			panic(readErr)
		}
		var (
			total = buff[:readSize]
			pos   = len(total) - 1
		)
		// 当最后一个字符是\n时，忽略这个\n
		if total[pos] == '\n' {
			pos--
		}
		for pos >= 0 && n > 0 {
			if total[pos] == '\n' {
				n--
			}
			if n == 0 {
				break
			}
			pos--
		}
		r := total[pos+1:]
		return unsafe.String(&r[0], len(r))
	} else {
		var (
			i      = now
			pos    = now
			result strings.Builder
		)

		for i >= 0 && n > 0 {
			i, ee = fd.Seek(-buffSize, io.SeekCurrent)
			if ee != nil {
				panic(ee)
			}
			// 读取这些字符到buff里
			_, ee = fd.ReadAt(buff, i)
			if ee != nil {
				panic(ee)
			}
			for j := buffSize - 1; j >= 0; j-- {
				if n < 0 {
					goto end
				} else if buff[j] == '\n' {
					n--
				}
				pos--
			}
		}
	end:
		pos += 1
		// 移动buffSize位，pos只可能 >= i,不可能小于i
		if pos > i {
			_, ee = fd.Seek(pos-i, io.SeekCurrent)
		}

		for {
			readSize, err := fd.Read(buff)
			if errors.Is(err, io.EOF) {
				break
			}
			result.Write(buff[:readSize])
		}
		return result.String()
	}
}
func countInt[T ~int | ~int64](i T) int {
	c := 1
	for i > 10 {
		i /= 10
		c++
	}
	return c
}

const (
	_ = 1 << (iota * 10)
	KB
	MB
	GB
	TB
)

func human(num int64, b bool) string {
	if !b || num < KB {
		return strconv.Itoa(int(num))
	}
	var (
		unit string
		n    = float64(num)
	)

	if n > KB && n < MB {
		unit = "K"
		n /= KB
	} else if n > MB && n < GB {
		unit = "M"
		n /= MB
	} else if n > GB && n < TB {
		unit = "G"
		n /= GB
	} else if n > TB {
		unit = "T"
		n /= TB
	}
	return fmt.Sprintf("%.1f%s", n, unit)
}

func Ls(p string) {
	ls(p, false)
}

// Ls 输出目录或者文件信息
func ls(p string, h bool) {
	var (
		format     = "%-10s\t%-16s\t%+7s\t%s\n"
		timeFormat = "2006/01/02 15:04"
	)
	state, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		panic(p + " 路径不存在")
	}
	fmt.Printf(format, "Mode", "ModTime", "Size", "Name")

	if !state.IsDir() {
		fmt.Printf(
			format,
			state.Mode(),
			state.ModTime().Format(timeFormat),
			human(state.Size(), h),
			state.Name(),
		)
		return
	} else {
		entrys, err := os.ReadDir(p)
		if err != nil {
			panic("读取目录内容失败: " + err.Error())
		}
		size := len(entrys)
		if size > 0 {
			sort.Slice(entrys, func(i, j int) bool {
				ii, _ := entrys[i].Info()
				jj, _ := entrys[j].Info()
				return ii.Size() < jj.Size()
			})
			if !h {
				ii, _ := entrys[size-1].Info()
				format = "%-10s\t%-16s\t%" + strconv.Itoa(countInt(ii.Size())) + "s\t%s\n"
			}
			for _, entry := range entrys {
				tp, ee := entry.Info()
				if ee == nil {
					fmt.Printf(
						format,
						tp.Mode(),
						tp.ModTime().Format(timeFormat),
						human(tp.Size(), h),
						tp.Name(),
					)
				}
			}
		}
	}
}
