package fn

import (
	"archive/zip"
	"bytes"
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
	return *(*string)(unsafe.Pointer(&b))
}

func ToBytes(s string) []byte {
	// 字符串结构头
	strHeader := (*[2]uintptr)(unsafe.Pointer(&s))
	// 切片结构头，最后一位为cap
	sliceHeader := [3]uintptr{strHeader[0], strHeader[1], strHeader[1]}
	return *(*[]byte)(unsafe.Pointer(&sliceHeader))
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

func ZipFile(source, target string) error {
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
