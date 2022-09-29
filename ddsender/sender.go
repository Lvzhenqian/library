package ddsender

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Lvzhenqian/library/fn"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	cacheData = &cacheSenderCountType{
		data: make(map[string]*limitAttribeType, 0),
		lock: new(sync.RWMutex),
	}

	ErrTimeFormat    = errors.New("格式不正确,例: 2021-11-16T16:20:49+08:00 ~ 2021-11-16T16:20:49+08:00")
	ErrIgnoreMessage = errors.New("message are ignored")
)

func sign(Secret string) (string, int64) {
	timestamp := time.Now().UnixNano() / 1e6
	toSign := fmt.Sprintf("%d\n%s", timestamp, Secret)
	mac := hmac.New(sha256.New, []byte(Secret))
	mac.Write([]byte(toSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), timestamp
}

func parseTime(s string) (start, end time.Time, err error) {
	list := strings.Split(s, "~")
	if len(list) != 2 {
		err = ErrTimeFormat
		return
	}
	start, err = time.Parse(time.RFC3339, strings.TrimSpace(list[0]))
	if err != nil {
		return
	}
	end, err = time.Parse(time.RFC3339, strings.TrimSpace(list[1]))
	return
}

type SenderType struct {
	// dingTalk 群机器人token
	Token string
	// dingTalk 群机器人secret
	Secret string
}

func NewSender(token, secret string) *SenderType {
	return &SenderType{
		Token:  token,
		Secret: secret,
	}
}

func (c *SenderType) httpClient(req *http.Request) ([]byte, error) {
	var client = http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *SenderType) Sender(Msg *Message) error {
	var (
		URI    = "https://oapi.dingtalk.com/robot/send"
		retMsg Response
	)
	msg, marshalErr := json.Marshal(Msg)
	if marshalErr != nil {
		return marshalErr
	}
	req, reqErr := http.NewRequest("POST", URI, bytes.NewReader(msg))
	if reqErr != nil {
		return reqErr
	}
	req.Header.Add("Content-Type", "application/json")
	query, _ := url.ParseQuery(req.URL.RawQuery)
	query.Add("access_token", c.Token)
	if c.Secret != "" {
		s, timestamp := sign(c.Secret)
		query.Add("sign", s)
		query.Add("timestamp", strconv.FormatInt(timestamp, 10))
	}
	req.URL.RawQuery = query.Encode()
	b, e := c.httpClient(req)
	if e != nil {
		return e
	}
	if err := json.Unmarshal(b, &retMsg); err != nil {
		return err
	}
	if retMsg.Errcode != 0 {
		return errors.New(fmt.Sprintf("code: %d,msg: %s", retMsg.Errcode, retMsg.Errmsg))
	}
	return nil
}

func (c *SenderType) MarkdownSender(Title, Msg string, atUser []string, atAll bool) error {
	return c.Sender(&Message{
		MsgType: Markdown,
		Markdown: MarkdownType{
			Title: Title,
			Text:  Msg,
		},
		At: AtType{
			IsAtAll:   atAll,
			AtMobiles: atUser,
		},
	})
}

type LimitSenderType struct {
	SenderType
	// 发送钉钉的间隔时间
	Interval time.Duration
	// 在 Interval 内总共发送多少次
	Limit int
	// 忽略的消息列表
	Ignore []IgnoreMessageType
}

func NewLimitSender(token, secret string, limit int, duration time.Duration, ignores []IgnoreMessageType) *LimitSenderType {
	return &LimitSenderType{
		SenderType: SenderType{
			Token:  token,
			Secret: secret,
		},
		Limit:    limit,
		Interval: duration,
		Ignore:   ignores,
	}
}

func (c *LimitSenderType) LimitSender(Title, Msg string) error {
	key := fn.Sha256sum([]byte(Msg))
	now := time.Now()

	for _, ignore := range c.Ignore {
		if fn.InSlice(ignore.MessageHash)(key) {
			now := time.Now()
			start, end, err := parseTime(ignore.Time)
			if err != nil {
				return err
			}
			if now.After(start) && now.Before(end) {
				return fmt.Errorf("%s %w", key, ErrIgnoreMessage)
			}
		}
	}

	count, exist := cacheData.GetCount(key)
	// 当 key 不存在，或者当前时间已经超过存储的时间时，重新生成这个值
	if !exist || count.expire.Before(now) {
		count = cacheData.Reset(key, now.Add(c.Interval))
	}
	// 当前次数少于限制次数,并且当前时间小于超时时间
	if count.total < c.Limit && count.expire.After(now) {
		n := cacheData.Add(key)
		body := fmt.Sprintf("%s\n\n%s total: %d", Msg, key, n)
		return c.MarkdownSender(Title, body, nil, true)
	}
	return nil
}
