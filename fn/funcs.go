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
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
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

func ZipFile(source, target string, level int) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	if path.IsAbs(source) {
		if err := os.Chdir(path.Dir(source)); err != nil {
			return err
		}
		source = path.Base(source)
	}
	defer zipFile.Close()
	archive := zip.NewWriter(zipFile)
	defer archive.Close()
	archive.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, level)
	})
	stat, statErr := os.Stat(source)
	if statErr != nil {
		return statErr
	}
	if stat.IsDir() {
		return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}
				header.Method = zip.Deflate
				header.Modified = time.Unix(info.ModTime().Unix(), 0)
				header.Name = path
				writer, err := archive.CreateHeader(header)
				if err != nil {
					return err
				}
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = io.Copy(writer, file)
				return err
			}
			return nil
		})
	}

	header, headErr := zip.FileInfoHeader(stat)
	if headErr != nil {
		return headErr
	}
	header.Method = zip.Deflate
	header.Modified = time.Unix(stat.ModTime().Unix(), 0)
	header.Name = source
	writer, createErr := archive.CreateHeader(header)
	if createErr != nil {
		return err
	}
	file, openErr := os.Open(source)
	if openErr != nil {
		return openErr
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
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
		result         = make([]string, n)
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
	if now < buffSize {
		buffSize = now
	}
	var buff = make([]byte, buffSize)

	// 最少输出1行
	if n == 0 {
		n = 1
	}

	for n > 0 && now > 0 {
		// 每次获取buffSize个字符
		now, ee = fd.Seek(-buffSize, io.SeekCurrent)
		if ee != nil {
			panic(ee)
		}
		// 读取这些字符到buff里
		_, ee = fd.ReadAt(buff, now)
		if ee != nil {
			panic(ee)
		}
		// 从buff 的最后第二个字符往前查找 \n 字符
		for i := buffSize - 2; i >= 0; i-- {
			if buff[i] == '\n' {
				// 当前char为\n, 当前行则为 i+1 到buffSize
				v := buff[i+1 : buffSize]
				tmp := make([]byte, len(v))
				copy(tmp, v)
				// 从后往前放
				n--
				result[n] = ToString(tmp)
				// 移动当前位置到 i 地址
				now, ee = fd.Seek(i, io.SeekCurrent)
				if ee != nil {
					panic(ee)
				}
				if now < buffSize {
					buffSize = now
				}
				break
			} else if i == 0 {
				n--
				result[n] = string(buff[i])
			}
		}
	}
	if n != 0 {
		return strings.Join(result[n:], "\n")
	}
	return strings.Join(result, "\n")
}

func countInt[T ~int | ~int64](i T) int {
	c := 1
	for i > 10 {
		i /= 10
		c++
	}
	return c
}

// Ls 输出目录或者文件信息
func Ls(p string) {
	state, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		panic(p + " 路径不存在")
	}
	fmt.Printf("%-10s\t%-16s\t%s\t%s\n", "Mode", "ModTime", "Size", "Name")

	if !state.IsDir() {
		fmt.Printf(
			"%-10s\t%-16s\t%d\t%s\n",
			state.Mode(),
			state.ModTime().Format("2006/01/02 15:04"),
			state.Size(),
			state.Name(),
		)
		return
	} else {
		entrys, err := os.ReadDir(p)
		if err != nil {
			panic("读取目录内容失败: " + err.Error())
		}

		sort.Slice(entrys, func(i, j int) bool {
			ii, _ := entrys[i].Info()
			jj, _ := entrys[j].Info()
			return ii.Size() > jj.Size()
		})
		ii, _ := entrys[0].Info()
		format := "%-10s\t%-16s\t%" + strconv.Itoa(countInt(ii.Size())) + "d\t%s\n"
		for _, entry := range entrys {

			tp, ee := entry.Info()
			if ee != nil {
				continue
			}
			fmt.Printf(
				format,
				tp.Mode(),
				tp.ModTime().Format("2006/01/02 15:04"),
				tp.Size(),
				tp.Name(),
			)
		}
	}
}
